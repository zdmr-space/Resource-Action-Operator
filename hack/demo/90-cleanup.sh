#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PATH="${ROOT_DIR}/bin:${PATH}"

CLUSTER_NAME="${CLUSTER_NAME:-rao-dev}"
DELETE_CLUSTER="${DELETE_CLUSTER:-false}"
HELM_RELEASE="${HELM_RELEASE:-resource-action-operator}"
HELM_NAMESPACE="${HELM_NAMESPACE:-resource-action-operator-system}"

echo "Cleaning demo resources..."
kubectl delete -f "${ROOT_DIR}/hack/demo/manifests/trigger-nginx-a.yaml" -n ra-demo --ignore-not-found=true || true
kubectl delete -f "${ROOT_DIR}/hack/demo/manifests/trigger-nginx-b.yaml" -n ra-demo --ignore-not-found=true || true
kubectl delete -f "${ROOT_DIR}/template-files/demo/resourceaction-deployment-http.yaml" --ignore-not-found=true || true
kubectl delete -f "${ROOT_DIR}/template-files/demo/resourceaction-deployment-tls.yaml" --ignore-not-found=true || true
kubectl delete -f "${ROOT_DIR}/hack/demo/manifests/nginx-tls.yaml" --ignore-not-found=true || true
kubectl delete -f "${ROOT_DIR}/hack/demo/manifests/nginx-http.yaml" --ignore-not-found=true || true
kubectl delete -f "${ROOT_DIR}/hack/demo/manifests/echo-backend.yaml" --ignore-not-found=true || true
kubectl -n ra-demo delete secret ra-nginx-tls ra-nginx-ca --ignore-not-found=true || true
kubectl delete -f "${ROOT_DIR}/hack/demo/manifests/namespace.yaml" --ignore-not-found=true || true

echo "Cleaning operator deployment and CRDs..."
helm uninstall "${HELM_RELEASE}" -n "${HELM_NAMESPACE}" || true
kubectl delete crd resourceactions.ops.yusaozdemir.de --ignore-not-found=true || true

if [[ "${DELETE_CLUSTER}" == "true" ]]; then
  echo "Deleting kind cluster '${CLUSTER_NAME}'..."
  kind delete cluster --name "${CLUSTER_NAME}" || true
fi

echo "Cleanup done."
