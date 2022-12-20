#!/usr/bin/env bash

gibo dump JetBrains Go Terraform >.gitignore
{
  # Ignore example Terraform lock file
  echo '.terraform.lock.hcl'

  # Ignore builds
  echo 'dist/'
  echo 'gcp2aws'
} >>.gitignore
