#!/usr/bin/env bash
#MISE depends = ["go:build"]

set -x
./build/cli --dir "${MISE_PROJECT_ROOT}/pb_data" $@
