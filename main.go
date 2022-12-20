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
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"golang.org/x/oauth2/google"
)

// Get OIDC ID token for Application Default Credential
// https://developers.google.com/identity/openid-connect/openid-connect#exchangecode
func getDefaultIdToken(ctx context.Context) (string, error) {
	ts, err := google.DefaultTokenSource(ctx, credentials.DefaultAuthScopes()...)
	if err != nil {
		return "", err
	}

	token, err := ts.Token()
	if err != nil {
		return "", err
	}

	return token.Extra("id_token").(string), nil
}

// Get OIDC ID token for a service account with impersonation
// You need "roles/iam.serviceAccountTokenCreator" role for your account.
// https://cloud.google.com/iam/docs/impersonating-service-accounts#allow-impersonation
func getImpersonatedIdToken(ctx context.Context, audience string, serviceAccountEmail string) (string, error) {
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

	err := json.NewDecoder(base64.NewDecoder(base64.URLEncoding, strings.NewReader(segments[1]))).Decode(&body)
	if err != nil {
		return "", err
	}

	return body.Email, nil
}

func assumeRole(ctx context.Context, roleArn string, roleSessionName string, token string, duration time.Duration) (*types.Credentials, error) {
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

func getCacheFilename(roleArn string) (string, error) {
	baseCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	cacheDir := baseCacheDir + "/gcp2aws"
	err = os.MkdirAll(cacheDir, 0700)
	if err != nil {
		return "", err
	}

	roleSum := sha256.Sum256([]byte(roleArn))
	filename := cacheDir + "/" + hex.EncodeToString(roleSum[:]) + ".json"
	return filename, nil
}

func writeToCache(roleArn string, data []byte) error {
	filename, err := getCacheFilename(roleArn)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, data, 0600)
	if err != nil {
		return err
	}

	return nil
}

func readFromCache(roleArn string) (string, error) {
	filename, err := getCacheFilename(roleArn)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	var cred TemporaryCredential
	err = json.Unmarshal(data, &cred)
	if err != nil {
		return "", err
	}

	if time.Now().After(cred.Expiration) {
		return "", errors.New("credential expired")
	}

	return string(data), nil
}

type TemporaryCredential struct {
	Version         int       `json:"Version"`
	AccessKeyId     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	SessionToken    string    `json:"SessionToken"`
	Expiration      time.Time `json:"Expiration"`
}

var (
	serviceAccountEmail string
	roleArn             string
	duration            time.Duration
)

func main() {
	flag.CommandLine.SetOutput(os.Stderr)
	flag.StringVar(&roleArn, "r", "", "Role ARN to AssumeRole")
	flag.DurationVar(&duration, "d", time.Hour, "Duration for a short-lived credential")
	flag.StringVar(&serviceAccountEmail, "i", "", "GCP Service account email to impersonate. If not specified, use Application Default Credential.")
	flag.Parse()

	log.SetOutput(os.Stderr)

	if len(roleArn) == 0 {
		log.Fatalln("Argument Required: -r <Role ARN>")
	}

	cache, err := readFromCache(roleArn)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(cache)
		os.Exit(0)
	}

	ctx := context.Background()

	defaultIdToken, err := getDefaultIdToken(ctx)
	if err != nil {
		log.Fatal(err)
	}

	email, err := extractEmailFromIdToken(defaultIdToken)
	if err != nil {
		log.Fatal(err)
	}

	var idToken = defaultIdToken
	if len(serviceAccountEmail) != 0 {
		idToken, err = getImpersonatedIdToken(ctx, email, serviceAccountEmail)
		if err != nil {
			log.Fatal(err)
		}
	}

	cred, err := assumeRole(ctx, roleArn, email, idToken, duration)
	if err != nil {
		log.Fatal(err)
	}

	// https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html
	out, err := json.MarshalIndent(TemporaryCredential{
		Version:         1,
		AccessKeyId:     *cred.AccessKeyId,
		SecretAccessKey: *cred.SecretAccessKey,
		SessionToken:    *cred.SessionToken,
		Expiration:      *cred.Expiration,
	}, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	err = writeToCache(roleArn, out)
	if err != nil {
		log.Println("Failed to write cache")
	}

	fmt.Println(string(out))
}
