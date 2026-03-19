#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

CLUSTER_NAME="${CLUSTER_NAME:-rao-dev}"
FULL_RESET="${FULL_RESET:-false}"

if [[ "${FULL_RESET}" == "true" ]]; then
  DELETE_CLUSTER=true CLUSTER_NAME="${CLUSTER_NAME}" "${ROOT_DIR}/hack/demo/90-cleanup.sh"
else
  "${ROOT_DIR}/hack/demo/90-cleanup.sh"
fi

"${ROOT_DIR}/hack/demo/00-kind-up.sh"
"${ROOT_DIR}/hack/demo/10-deploy-operator-kind.sh"
"${ROOT_DIR}/hack/demo/20-apply-receivers.sh"
"${ROOT_DIR}/hack/demo/30-apply-resourceactions.sh"

echo "Reset complete. Run triggers with:"
echo "  ${ROOT_DIR}/hack/demo/40-trigger-actions.sh"
