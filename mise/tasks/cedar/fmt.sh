#!/usr/bin/env bash
#MISE description="Format Cedar policy files"

set -euo pipefail

cedar format -p policies/policy.cedar --write
