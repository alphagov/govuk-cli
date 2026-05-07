#!/usr/bin/env bash

set -e
rm -rf completions
mkdir completions
go build
for sh in bash zsh fish; do
  ./govuk-cli completion "$sh" > "completions/govuk-cli.$sh"
done
