#!/usr/bin/env bash

gibo dump JetBrains Go Terraform >.gitignore
{
  # Ignore secrets
  echo 'secret.env'

  # Ignore example Terraform lock file
  echo '.terraform.lock.hcl'

  # Ignore builds
  echo 'dist/'
  echo 'gcp2aws'

  # Ignore coverage files
  echo 'cover.out'
  echo 'cover.html'
} >>.gitignore
