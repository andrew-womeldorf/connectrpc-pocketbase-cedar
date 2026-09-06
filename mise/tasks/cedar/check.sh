#!/usr/bin/env bash
#MISE description="Check Cedar policies parse correctly and are formatted"

set -euo pipefail

echo "Checking Cedar policies parse correctly..."
cedar check-parse -p policies/policy.cedar -s policies/schema.cedarschema
echo "Policies parse OK."

echo ""
echo "Checking Cedar policy formatting..."
if cedar format -p policies/policy.cedar --check > /dev/null; then
    echo "Formatting OK."
else
    echo "Formatting check failed. Run 'mise run cedar:fmt' to fix."
    exit 1
fi
