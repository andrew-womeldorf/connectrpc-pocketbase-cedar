#!/usr/bin/env bash
#MISE depends = ["proto:generate"]
#MISE hide=true

go fmt ./...
go mod tidy
goimports -w .
