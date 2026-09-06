#!/usr/bin/env bash
#MISE description="Run Cedar authorization test cases"
#MISE depends=["cedar:validate"]

set -euo pipefail

cedar run-tests \
    -p policies/policy.cedar \
    -s policies/schema.cedarschema \
    --tests policies/tests.json
