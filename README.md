# gcp2aws
AWS credential helper for GCP.

[![Go Report Card](https://goreportcard.com/badge/github.com/porkbeans/gcp2aws)](https://goreportcard.com/report/github.com/porkbeans/gcp2aws)
[![Build & Test](https://github.com/porkbeans/gcp2aws/actions/workflows/test.yml/badge.svg)](https://github.com/porkbeans/gcp2aws/actions/workflows/test.yml)
[![Maintainability](https://api.codeclimate.com/v1/badges/c8a14b2dd09e72725014/maintainability)](https://codeclimate.com/github/porkbeans/gcp2aws/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/c8a14b2dd09e72725014/test_coverage)](https://codeclimate.com/github/porkbeans/gcp2aws/test_coverage)

# Requirements
- GCP
  - Service Accounts that allow you to impersonate(`roles/iam.serviceAccountTokenCreator`)
- AWS
  - IAM Roles that allow service accounts to `sts:AssumeRoleWithWebIdentity`

# Installation
```
go install github.com/porkbeans/gcp2aws@latest
```

# Usage

```text
SYNOPSIS
    gcp2aws -i <SERVICE ACCOUNT EMAIL> -r <ROLE ARN> [-d <DURATION>]

OPTIONS
    -i <SERVICE ACCOUNT EMAIL>
        Service account email to impersonate.
    -r <ROLE ARN>
        Role ARN to AssumeRoleWithWebIdentity.
    -d <DURATION>
        Credential duration. Accept format for Go time.ParseDuration.
        See https://pkg.go.dev/time#ParseDuration
```

# Examples
See [Example Terraform](./example/main.tf) to set up GCP Service Account and AWS IAM Role.

AssumeRole with impersonated GCP service account identity.

`~/.aws/config`
```text
[profile example]
credential_process = /path/to/gcp2aws -r <ROLE ARN> -i <SERVICE ACCOUNT EMAIL>
region = <YOUR REGION>
```

# Development

## Required tools
- [go](https://go.dev/) for compiling and testing
- [GNU make](https://www.gnu.org/software/make/) for task runner
- [direnv](https://github.com/direnv/direnv) for loading environment variables for tests
- [gibo](https://github.com/simonwhitaker/gibo) for updating .gitignore boilerplate

## Preparing
- `cp example.env secret.env` and edit each values in `secret.env` for your test environment.
- `direnv allow`
- `make test` to confirm that you can run tests

# Similar projects
- https://github.com/doitintl/janus
