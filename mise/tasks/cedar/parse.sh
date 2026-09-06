#!/usr/bin/env bash
#MISE hide=true
#MISE description="Check Cedar policies parse correctly"

set -euo pipefail

cedar check-parse -p policies/policy.cedar -s policies/schema.cedarschema
