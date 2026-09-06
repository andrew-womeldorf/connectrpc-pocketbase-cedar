#!/usr/bin/env bash
#MISE sources = ["**/*.go", "policies/*"]

mkdir -p build
go build -o build/cli main.go
