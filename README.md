# gcp2aws
AWS credential helper for GCP.

[![Go Report Card](https://goreportcard.com/badge/github.com/porkbeans/gcp2aws)](https://goreportcard.com/report/github.com/porkbeans/gcp2aws)
[![Build](https://github.com/porkbeans/gcp2aws/actions/workflows/build.yml/badge.svg)](https://github.com/porkbeans/gcp2aws/actions/workflows/build.yml)

# Requirements
- GCP
  - Service Accounts that allow you to impersonate(`roles/iam.serviceAccountTokenCreator`)
- AWS
  - IAM Roles that allow service accounts to `sts:AssumeRoleWithWebIdentity`

# Usage

```text
SYNOPSIS
    gcp2aws -r <ROLE ARN> [-i <SERVICE ACCOUNT EMAIL>] [-d <DURATION>]

OPTIONS
    -r <ROLE_ARN>
        Role ARN to AssumeRoleWithWebIdentity.
    -i <SERVICE ACCOUNT EMAIL>
        Service account email to impersonate.
    -d <DURATION>
        Credential duration. Accept format for Go time.ParseDuration.
        See https://pkg.go.dev/time#ParseDuration
```

# Examples
See [Example Terraform](./example/main.tf) to set up GCP Service Account and AWS IAM Role.

## AssumeRole with application default credential
`~/.aws/config`
```text
[example]
credential_process = /path/to/gcp2aws -r <ROLE ARN>
region = <YOUR REGION>
```

## AssumeRole with impersonated GCP service account identity
`~/.aws/config`
```text
[example]
credential_process = /path/to/gcp2aws -r <ROLE ARN> -i <SERVICE ACCOUNT EMAIL>
region = <YOUR REGION>
```
