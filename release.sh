#!/usr/bin/env bash

set -x

RELEASE=0.2
mkdir -p build/
env GOOS=darwin GOARCH=amd64 go build -o build/anki-helper-v${RELEASE}-darwin-amd64 .
env GOOS=darwin GOARCH=arm64 go build -o build/anki-helper-v${RELEASE}-darwin-arm64 .
env GOOS=linux GOARCH=amd64 go build -o build/anki-helper-v${RELEASE}-linux-amd64 .
env GOOS=windows GOARCH=amd64 go build -o build/anki-helper-v${RELEASE}-windows-amd64.exe .