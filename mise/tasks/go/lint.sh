#!/usr/bin/env bash
#MISE depends = ["proto:generate"]
#MISE wait_for = ["go:format"]
#MISE hide=true

golangci-lint run ./...
