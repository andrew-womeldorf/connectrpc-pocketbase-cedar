#!/usr/bin/env bash
#MISE description = "Pre-compress static assets with gzip"
#MISE hide = true
#MISE sources = ["internal/web/static/*.css"]
set -euo pipefail

STATIC_DIR="internal/web/static"

find "$STATIC_DIR" -type f \
  ! -name '*.gz' ! -name '*.br' ! -name '*.zst' \
  -print0 | while IFS= read -r -d '' f; do
    gzip -9kf "$f"
    echo "gzip: $f → ${f}.gz"
done
