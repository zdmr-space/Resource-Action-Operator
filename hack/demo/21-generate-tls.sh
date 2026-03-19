#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PATH="${ROOT_DIR}/bin:${PATH}"

NAMESPACE="${NAMESPACE:-ra-demo}"
SERVICE_NAME="${SERVICE_NAME:-ra-nginx-tls}"
SERVER_NAME="${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

cat > "${TMP_DIR}/server.cnf" <<EOF
[ req ]
distinguished_name = dn
req_extensions = req_ext
prompt = no

[ dn ]
CN = ${SERVER_NAME}

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = ${SERVICE_NAME}.${NAMESPACE}.svc
DNS.2 = ${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local
EOF

openssl genrsa -out "${TMP_DIR}/ca.key" 2048
openssl req -x509 -new -nodes -key "${TMP_DIR}/ca.key" -sha256 -days 3650 \
  -subj "/CN=ra-demo-ca" \
  -out "${TMP_DIR}/ca.crt"

openssl genrsa -out "${TMP_DIR}/server.key" 2048
openssl req -new -key "${TMP_DIR}/server.key" \
  -out "${TMP_DIR}/server.csr" \
  -config "${TMP_DIR}/server.cnf"

openssl x509 -req \
  -in "${TMP_DIR}/server.csr" \
  -CA "${TMP_DIR}/ca.crt" \
  -CAkey "${TMP_DIR}/ca.key" \
  -CAcreateserial \
  -out "${TMP_DIR}/server.crt" \
  -days 3650 \
  -sha256 \
  -extensions req_ext \
  -extfile "${TMP_DIR}/server.cnf"

kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

kubectl -n "${NAMESPACE}" create secret tls ra-nginx-tls \
  --cert="${TMP_DIR}/server.crt" \
  --key="${TMP_DIR}/server.key" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n "${NAMESPACE}" create secret generic ra-nginx-ca \
  --from-file=ca.crt="${TMP_DIR}/ca.crt" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "Created TLS secrets in namespace '${NAMESPACE}'"
echo "- secret/tls: ra-nginx-tls"
echo "- secret/ca:  ra-nginx-ca"
