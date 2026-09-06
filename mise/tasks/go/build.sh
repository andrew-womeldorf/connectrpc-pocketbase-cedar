#!/usr/bin/env bash
#MISE hide = true
#MISE depends = ["proto:generate"]
#MISE sources = ["**/*.go", "policies/*"]

mkdir -p build
go build -o build/cli main.go
