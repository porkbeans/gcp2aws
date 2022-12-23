package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

// Get OIDC ID token for a service account with impersonation
// You need "roles/iam.serviceAccountTokenCreator" role for your account.
// https://cloud.google.com/iam/docs/impersonating-service-accounts#allow-impersonation
func getIdToken(audience string, serviceAccountEmail string) (string, error) {
	ctx := context.Background()
	gcpIamClient, err := credentials.NewIamCredentialsClient(ctx)
	if err != nil {
		return "", err
	}

	// https://pkg.go.dev/cloud.google.com/go/iam/credentials/apiv1/credentialspb#GenerateIdTokenRequest
	req := credentialspb.GenerateIdTokenRequest{
		Name:         "projects/-/serviceAccounts/" + serviceAccountEmail,
		Audience:     audience,
		IncludeEmail: true,
	}

	// https://cloud.google.com/iam/docs/create-short-lived-credentials-direct#sa-credentials-oidc
	resp, err := gcpIamClient.GenerateIdToken(ctx, &req)
	if err != nil {
		return "", err
	}

	return resp.Token, nil
}

type jwtBody struct {
	Email string `json:"email"`
}

func extractEmailFromIdToken(idToken string) (string, error) {
	segments := strings.Split(idToken, ".")
	body := jwtBody{}

	err := json.NewDecoder(base64.NewDecoder(base64.RawURLEncoding, strings.NewReader(segments[1]))).Decode(&body)
	if err != nil {
		return "", err
	}

	return body.Email, nil
}

func assumeRole(roleArn string, roleSessionName string, token string, duration time.Duration) (*types.Credentials, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	awsStsClient := sts.NewFromConfig(cfg)

	durationSeconds := int32(duration.Seconds())

	req := sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          &roleArn,
		RoleSessionName:  &roleSessionName,
		WebIdentityToken: &token,
		DurationSeconds:  &durationSeconds,
	}

	resp, err := awsStsClient.AssumeRoleWithWebIdentity(ctx, &req)
	if err != nil {
		return nil, err
	}

	return resp.Credentials, nil
}

type TemporaryCredential struct {
	Version         int       `json:"Version"`
	AccessKeyId     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	SessionToken    string    `json:"SessionToken"`
	Expiration      time.Time `json:"Expiration"`
}

func getAwsCredential(serviceAccountEmail string, roleArn string, duration time.Duration, cred *TemporaryCredential) error {
	idToken, err := getIdToken("gcp2aws", serviceAccountEmail)
	if err != nil {
		return err
	}

	email, err := extractEmailFromIdToken(idToken)
	if err != nil {
		return err
	}

	resp, err := assumeRole(roleArn, email, idToken, duration)
	if err != nil {
		return err
	}

	// https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html
	cred.Version = 1
	cred.AccessKeyId = *resp.AccessKeyId
	cred.SecretAccessKey = *resp.SecretAccessKey
	cred.SessionToken = *resp.SessionToken
	cred.Expiration = *resp.Expiration

	return nil
}

func getCacheFilename(roleArn string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	roleSum := sha256.Sum256([]byte(roleArn))
	filename := cacheDir + "/gcp2aws/" + hex.EncodeToString(roleSum[:]) + ".json"
	return filename, nil
}

func writeToCache(roleArn string, cred TemporaryCredential) error {
	data, err := json.Marshal(cred)
	if err != nil {
		return err
	}

	filename, err := getCacheFilename(roleArn)
	if err != nil {
		return err
	}

	_ = os.Mkdir(path.Dir(filename), 0700)
	err = os.WriteFile(filename, data, 0600)
	if err != nil {
		return err
	}

	return nil
}

func readFromCache(roleArn string, cred *TemporaryCredential) error {
	filename, err := getCacheFilename(roleArn)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, cred)
	if err != nil {
		return err
	}

	if time.Now().After(cred.Expiration) {
		err = errors.New("credential expired")
	}

	return err
}

var (
	serviceAccountEmail string
	roleArn             string
	duration            time.Duration
	quiet               bool
)

func init() {
	flag.StringVar(&serviceAccountEmail, "i", os.Getenv("GCP2AWS_GCP_SERVICE_ACCOUT_EMAIL"), "GCP Service account email to impersonate. If not specified, use Application Default Credential.")
	flag.StringVar(&roleArn, "r", os.Getenv("GCP2AWS_AWS_ROLE_ARN"), "Role ARN to AssumeRole")
	flag.DurationVar(&duration, "d", time.Hour, "Duration for a short-lived credential")
	flag.BoolVar(&quiet, "q", false, "Suppress output.")

	flag.CommandLine.SetOutput(os.Stderr)

	log.SetOutput(os.Stderr)
}

func exec() int {
	flag.Parse()

	if len(serviceAccountEmail) == 0 {
		log.Println("Argument Required: -i <SERVICE ACCOUNT EMAIL>")
		return 1
	}

	if len(roleArn) == 0 {
		log.Println("Argument Required: -r <ROLE ARN>")
		return 1
	}

	var out io.Writer = os.Stdout
	if quiet {
		out = io.Discard
	}

	cred := TemporaryCredential{}

	err := readFromCache(roleArn, &cred)
	if err != nil {
		log.Println(err)
	} else {
		cacheJson, _ := json.Marshal(cred)
		_, _ = fmt.Fprintln(out, string(cacheJson))
		return 0
	}

	err = getAwsCredential(serviceAccountEmail, roleArn, duration, &cred)
	if err != nil {
		log.Println(err)
		return 1
	}

	_ = writeToCache(roleArn, cred)

	credJson, _ := json.Marshal(cred)
	_, _ = fmt.Fprintln(out, string(credJson))

	return 0
}

func main() {
	os.Exit(exec())
}
