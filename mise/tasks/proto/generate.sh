#!/usr/bin/env bash
#MISE description = "Rebuild the gen/ dir"
#MISE sources = ["proto/**/*.proto"]

rm -rf gen/
buf generate
