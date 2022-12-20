#!/usr/bin/env bash

gibo dump JetBrains Go Terraform >.gitignore
echo '.terraform.lock.hcl' >>.gitignore
