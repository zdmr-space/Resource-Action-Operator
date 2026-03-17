#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PATH="${ROOT_DIR}/bin:${PATH}"
HELM_NAMESPACE="${HELM_NAMESPACE:-resource-action-operator-system}"

OPERATOR_DEPLOYMENT="$(kubectl -n "${HELM_NAMESPACE}" get deploy -l control-plane=controller-manager -o jsonpath='{.items[0].metadata.name}')"

echo "==== Operator logs (last 200) ===="
if [[ -n "${OPERATOR_DEPLOYMENT}" ]]; then
  kubectl -n "${HELM_NAMESPACE}" logs deploy/"${OPERATOR_DEPLOYMENT}" -c manager --tail=200 || true
else
  echo "No operator deployment found in namespace '${HELM_NAMESPACE}'"
fi

echo
echo "==== Echo backend logs (method, headers, body) ===="
kubectl -n ra-demo logs deploy/ra-echo --tail=200 || true

echo
echo "==== NGINX HTTP access log ===="
HTTP_POD="$(kubectl -n ra-demo get pods -l app=ra-nginx-http -o jsonpath='{.items[0].metadata.name}')"
kubectl -n ra-demo exec "${HTTP_POD}" -- sh -c "tail -n 200 /var/log/nginx/access.log" || true

echo
echo "==== NGINX TLS access log ===="
TLS_POD="$(kubectl -n ra-demo get pods -l app=ra-nginx-tls -o jsonpath='{.items[0].metadata.name}')"
kubectl -n ra-demo exec "${TLS_POD}" -- sh -c "tail -n 200 /var/log/nginx/access.log" || true
