#!/usr/bin/env bash
#MISE description="Validate Cedar policies against the schema"

set -euo pipefail

cedar validate -p policies/policy.cedar -s policies/schema.cedarschema
