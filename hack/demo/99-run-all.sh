#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

"${ROOT_DIR}/hack/demo/00-kind-up.sh"
"${ROOT_DIR}/hack/demo/10-deploy-operator-kind.sh"
"${ROOT_DIR}/hack/demo/20-apply-receivers.sh"
"${ROOT_DIR}/hack/demo/30-apply-resourceactions.sh"
"${ROOT_DIR}/hack/demo/40-trigger-actions.sh"
"${ROOT_DIR}/hack/demo/50-show-logs.sh"
