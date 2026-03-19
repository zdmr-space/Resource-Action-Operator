#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PATH="${ROOT_DIR}/bin:${PATH}"

echo "Create trigger-nginx-a"
kubectl apply -f "${ROOT_DIR}/hack/demo/manifests/trigger-nginx-a.yaml"
kubectl -n ra-demo rollout status deploy/trigger-nginx-a --timeout=120s
sleep 3

echo "Delete trigger-nginx-a"
kubectl -n ra-demo delete -f "${ROOT_DIR}/hack/demo/manifests/trigger-nginx-a.yaml" --ignore-not-found=true
sleep 3

echo "Create trigger-nginx-b"
kubectl apply -f "${ROOT_DIR}/hack/demo/manifests/trigger-nginx-b.yaml"
kubectl -n ra-demo rollout status deploy/trigger-nginx-b --timeout=120s
sleep 3

echo "Delete trigger-nginx-b"
kubectl -n ra-demo delete -f "${ROOT_DIR}/hack/demo/manifests/trigger-nginx-b.yaml" --ignore-not-found=true
sleep 3

echo "ResourceAction status:"
kubectl -n ra-demo get resourceactions
kubectl -n ra-demo get resourceaction deployment-http-hook -o yaml | sed -n '1,220p'
kubectl -n ra-demo get resourceaction deployment-tls-hook -o yaml | sed -n '1,220p'
