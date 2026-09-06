#!/usr/bin/env bash
#MISE sources = ["**/*.go"]

mkdir -p build
go build -o build/cli main.go
