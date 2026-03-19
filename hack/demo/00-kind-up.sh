#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PATH="${ROOT_DIR}/bin:${PATH}"

CLUSTER_NAME="${CLUSTER_NAME:-rao-dev}"
KIND_CONFIG="${ROOT_DIR}/hack/demo/kind-config.yaml"

if kind get clusters | grep -qx "${CLUSTER_NAME}"; then
  echo "kind cluster '${CLUSTER_NAME}' already exists"
else
  kind create cluster --name "${CLUSTER_NAME}" --config "${KIND_CONFIG}"
fi

kubectl cluster-info --context "kind-${CLUSTER_NAME}"
kubectl get nodes -o wide
