package main

import (
	"encoding/base64"
	"flag"
	"os"
	"testing"
	"time"
)

var (
	AwsRoleArnForTest            = os.Getenv("GCP2AWS_AWS_ROLE_ARN")
	GcpServiceAccoutEmailForTest = os.Getenv("GCP2AWS_GCP_SERVICE_ACCOUT_EMAIL")
)

func TestGetDefaultIdToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, err := getDefaultIdToken()
		if err != nil {
			t.Log(err)
			t.Fail()
		}
	})

	t.Run("fail", func(t *testing.T) {
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/tmp/notfound.json")
		_, err := getDefaultIdToken()
		if err == nil {
			t.Fail()
		}
		t.Log(err)
	})
}

func TestGetImpersonatedIdToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, err := getImpersonatedIdToken("gcp2aws", GcpServiceAccoutEmailForTest)
		if err != nil {
			t.Log(err)
			t.Fail()
		}
	})

	t.Run("fail", func(t *testing.T) {
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/tmp/notfound.json")
		_, err := getImpersonatedIdToken("", "invalid@example.com")
		if err == nil {
			t.Log(err)
			t.Fail()
		}
	})

	t.Run("fail", func(t *testing.T) {
		_, err := getImpersonatedIdToken("", "invalid@example.com")
		if err == nil {
			t.Log(err)
			t.Fail()
		}
	})
}

func mockJwt(body string) string {
	return base64.URLEncoding.EncodeToString([]byte("header")) + "." +
		base64.URLEncoding.EncodeToString([]byte(body)) + "." +
		base64.URLEncoding.EncodeToString([]byte("signature"))
}

func TestExtractEmailFromIdToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		email, err := extractEmailFromIdToken(mockJwt("{\"email\":\"test@example.com\"}"))
		if err != nil {
			t.Log(err)
			t.Fail()
		}
		t.Log(email)
	})

	t.Run("fail", func(t *testing.T) {
		email, err := extractEmailFromIdToken(mockJwt("invalid json"))
		if err == nil {
			t.Log(email)
			t.Fail()
		}
		t.Log(err)
	})
}

func clearCache(roleArn string) {
	filename, _ := getCacheFilename(roleArn)
	_ = os.Remove(filename)
}

func mockCredential() TemporaryCredential {
	return TemporaryCredential{
		Version:         1,
		AccessKeyId:     "mock key id",
		SecretAccessKey: "mock key",
		SessionToken:    "mock token",
		Expiration:      time.Now().Add(time.Hour),
	}
}

func TestWriteToCache(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		clearCache(AwsRoleArnForTest)
		err := writeToCache(AwsRoleArnForTest, mockCredential())
		if err != nil {
			t.Log(err)
			t.Fail()
		}
	})

	t.Run("fail with env vars", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		t.Setenv("HOME", "")

		clearCache(AwsRoleArnForTest)
		err := writeToCache(AwsRoleArnForTest, mockCredential())
		if err.Error() != "neither $XDG_CACHE_HOME nor $HOME are defined" {
			t.Fail()
		}
		t.Log(err)
	})

	t.Run("fail with env vars", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "/root/.cache")

		clearCache(AwsRoleArnForTest)
		err := writeToCache(AwsRoleArnForTest, mockCredential())
		if err == nil {
			t.Fail()
		}
		t.Log(err)
	})
}

func TestReadFromCache(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		clearCache(AwsRoleArnForTest)
		_ = writeToCache(roleArn, mockCredential())

		cred := TemporaryCredential{}
		err := readFromCache(AwsRoleArnForTest, &cred)
		if err != nil {
			t.Log(err)
			t.Fail()
		}
	})

	t.Run("fail with invalid json", func(t *testing.T) {
		clearCache(AwsRoleArnForTest)
		filename, _ := getCacheFilename(AwsRoleArnForTest)
		_ = os.WriteFile(filename, []byte("invalid json"), 0600)

		cred := TemporaryCredential{}
		err := readFromCache(AwsRoleArnForTest, &cred)
		if err == nil {
			t.Fail()
		}
		t.Log(err)
	})

	t.Run("fail with env vars", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		t.Setenv("HOME", "")

		clearCache(AwsRoleArnForTest)
		cred := TemporaryCredential{}
		err := readFromCache(AwsRoleArnForTest, &cred)
		if err.Error() != "neither $XDG_CACHE_HOME nor $HOME are defined" {
			t.Fail()
		}
		t.Log(err)
	})

	t.Run("fail with env vars", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "/root/.cache")

		clearCache(AwsRoleArnForTest)
		cred := TemporaryCredential{}
		err := readFromCache(AwsRoleArnForTest, &cred)
		if err == nil {
			t.Fail()
		}
		t.Log(err)
	})
}

func TestExec(t *testing.T) {
	t.Run("success without cache", func(t *testing.T) {
		clearCache(AwsRoleArnForTest)

		_ = flag.Set("i", GcpServiceAccoutEmailForTest)
		_ = flag.Set("r", AwsRoleArnForTest)
		_ = flag.Set("d", "1h")
		_ = flag.Set("q", "true")
		if exec() != 0 {
			t.Fail()
		}
	})

	t.Run("success with valid cache", func(t *testing.T) {
		_ = flag.Set("i", GcpServiceAccoutEmailForTest)
		_ = flag.Set("r", AwsRoleArnForTest)
		_ = flag.Set("d", "1h")
		_ = flag.Set("q", "true")
		if exec() != 0 {
			t.Fail()
		}
	})

	t.Run("success with expired cache", func(t *testing.T) {
		cred := mockCredential()
		cred.Expiration = time.Now().Add(-6 * time.Hour)
		_ = writeToCache(AwsRoleArnForTest, cred)

		_ = flag.Set("i", GcpServiceAccoutEmailForTest)
		_ = flag.Set("r", AwsRoleArnForTest)
		_ = flag.Set("d", "1h")
		_ = flag.Set("q", "true")
		if exec() != 0 {
			t.Fail()
		}
	})

	t.Run("fail with invalid gcp credential path", func(t *testing.T) {
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/tmp/notfound.json")
		clearCache(AwsRoleArnForTest)

		_ = flag.Set("i", GcpServiceAccoutEmailForTest)
		_ = flag.Set("r", AwsRoleArnForTest)
		_ = flag.Set("d", "1h")
		if exec() != 1 {
			t.Fail()
		}
	})

	t.Run("fail with lack of role arn", func(t *testing.T) {
		clearCache(AwsRoleArnForTest)

		_ = flag.Set("i", "")
		_ = flag.Set("r", "")
		_ = flag.Set("d", "")
		if exec() != 1 {
			t.Fail()
		}
	})

	t.Run("fail with invalid role arn", func(t *testing.T) {
		clearCache(AwsRoleArnForTest)

		_ = flag.Set("i", GcpServiceAccoutEmailForTest)
		_ = flag.Set("r", "invalid role arn")
		_ = flag.Set("d", "1h")
		if exec() != 1 {
			t.Fail()
		}
	})

	t.Run("fail with invalid service account", func(t *testing.T) {
		clearCache(AwsRoleArnForTest)

		_ = flag.Set("i", "invalid@example.com")
		_ = flag.Set("r", AwsRoleArnForTest)
		_ = flag.Set("d", "1h")
		if exec() != 1 {
			t.Fail()
		}
	})

	t.Run("fail with invalid duration", func(t *testing.T) {
		clearCache(AwsRoleArnForTest)

		_ = flag.Set("i", GcpServiceAccoutEmailForTest)
		_ = flag.Set("r", AwsRoleArnForTest)
		_ = flag.Set("d", "1s")
		if exec() != 1 {
			t.Fail()
		}
	})
}
