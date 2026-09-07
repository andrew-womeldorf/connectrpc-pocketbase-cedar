#!/usr/bin/env bash
#MISE hide = true
#MISE depends = ["proto:generate", "web:compress", "web:hash"]
#MISE sources = ["**/*.go", "policies/*"]

mkdir -p build
go build -o build/cli main.go
