#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export PATH="${ROOT_DIR}/bin:${PATH}"

echo "Installing CRDs into current cluster context"
make -C "${ROOT_DIR}" install

echo "Starting operator locally (Ctrl+C to stop)"
make -C "${ROOT_DIR}" run
