#!/usr/bin/env bash
#MISE hide = true
#MISE description="Check Cedar policies are formatted"

set -euo pipefail
cedar format -p policies/policy.cedar --check > /dev/null
