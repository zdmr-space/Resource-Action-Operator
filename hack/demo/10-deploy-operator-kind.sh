#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PATH="${ROOT_DIR}/bin:${PATH}"

CLUSTER_NAME="${CLUSTER_NAME:-rao-dev}"
IMG="${IMG:-resource-action-operator:dev}"
HELM_RELEASE="${HELM_RELEASE:-resource-action-operator}"
HELM_NAMESPACE="${HELM_NAMESPACE:-resource-action-operator-system}"

if [[ "${IMG##*/}" == *:* ]]; then
  IMAGE_REPOSITORY="${IMG%:*}"
  IMAGE_TAG="${IMG##*:}"
else
  IMAGE_REPOSITORY="${IMG}"
  IMAGE_TAG="latest"
fi

echo "Building operator image: ${IMG}"
make -C "${ROOT_DIR}" docker-build IMG="${IMG}"

echo "Loading image into kind cluster: ${CLUSTER_NAME}"
kind load docker-image "${IMG}" --name "${CLUSTER_NAME}"

echo "Installing CRDs and deploying operator via Helm"
helm upgrade --install "${HELM_RELEASE}" "${ROOT_DIR}/charts/resource-action-operator" \
  --namespace "${HELM_NAMESPACE}" \
  --create-namespace \
  --set image.repository="${IMAGE_REPOSITORY}" \
  --set image.tag="${IMAGE_TAG}" \
  --set image.pullPolicy=IfNotPresent

OPERATOR_DEPLOYMENT="$(kubectl -n "${HELM_NAMESPACE}" get deploy -l control-plane=controller-manager -o jsonpath='{.items[0].metadata.name}')"
if [[ -z "${OPERATOR_DEPLOYMENT}" ]]; then
  echo "Could not find operator deployment in namespace '${HELM_NAMESPACE}'"
  exit 1
fi
kubectl -n "${HELM_NAMESPACE}" rollout status deploy/"${OPERATOR_DEPLOYMENT}" --timeout=180s
kubectl -n "${HELM_NAMESPACE}" get pods -o wide
