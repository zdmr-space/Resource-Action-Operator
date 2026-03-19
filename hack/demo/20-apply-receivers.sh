#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PATH="${ROOT_DIR}/bin:${PATH}"

kubectl apply -f "${ROOT_DIR}/hack/demo/manifests/namespace.yaml"
"${ROOT_DIR}/hack/demo/21-generate-tls.sh"

kubectl apply -f "${ROOT_DIR}/hack/demo/manifests/echo-backend.yaml"
kubectl apply -f "${ROOT_DIR}/hack/demo/manifests/nginx-http.yaml"
kubectl apply -f "${ROOT_DIR}/hack/demo/manifests/nginx-tls.yaml"

kubectl -n ra-demo rollout status deploy/ra-echo --timeout=180s
kubectl -n ra-demo rollout status deploy/ra-nginx-http --timeout=180s
kubectl -n ra-demo rollout status deploy/ra-nginx-tls --timeout=180s

kubectl -n ra-demo get pods -o wide
kubectl -n ra-demo get svc
