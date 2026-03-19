#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PATH="${ROOT_DIR}/bin:${PATH}"

kubectl apply -f "${ROOT_DIR}/template-files/demo/resourceaction-deployment-http.yaml"
kubectl apply -f "${ROOT_DIR}/template-files/demo/resourceaction-deployment-tls.yaml"

kubectl -n ra-demo get resourceactions
