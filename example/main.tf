terraform {
  required_version = "~> 1.3.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

variable "gcp_project" {
  type = string
}

variable "aws_profile" {
  type = string
}

variable "gmail_address" {
  type = string
}

provider "google" {
  project = var.gcp_project
}

provider "aws" {
  profile = var.aws_profile
}

resource "google_service_account" "example" {
  account_id   = "example"
  display_name = "Example service account"
}

# Allows an Gmail account to impersonate an service account.
# https://cloud.google.com/iam/docs/impersonating-service-accounts#allow-impersonation
data "google_iam_policy" "example" {
  binding {
    role    = "roles/iam.serviceAccountTokenCreator"
    members = [
      "user:${var.gmail_address}"
    ]
  }
}

resource "google_service_account_iam_policy" "example" {
  service_account_id = google_service_account.example.name
  policy_data        = data.google_iam_policy.example.policy_data
}

resource "aws_iam_role" "example" {
  name                 = "ExampleRole"
  path                 = "/"
  max_session_duration = "3600"
  managed_policy_arns  = ["arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"]

  # Allows an service account to assume role.
  # https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_iam-condition-keys.html#condition-keys-wif
  # https://developers.google.com/identity/openid-connect/openid-connect#an-id-tokens-payload
  assume_role_policy = jsonencode(
    {
      "Statement" : [
        {
          "Action" : "sts:AssumeRoleWithWebIdentity",
          "Effect" : "Allow",
          "Principal" : {
            "Federated" : "accounts.google.com"
          },
          "Condition" : {
            "StringEquals" : {
              "accounts.google.com:aud" : google_service_account.example.unique_id
            }
          }
        }
      ],
      "Version" : "2012-10-17"
    }
  )
}

output "gcp_service_account" {
  value = google_service_account.example.email
}

output "aws_role_arn" {
  value = aws_iam_role.example.arn
}
