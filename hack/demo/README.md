# Local Demo And Integration Setup

This folder contains a repeatable local setup for end-to-end manual tests with:

- kind cluster
- operator deployment
- HTTP and TLS NGINX gateways
- echo backend for request inspection (method, headers, body)
- ResourceAction templates for Create/Delete triggers
- helper scripts for apply, trigger, and logs

## Prerequisites

- `docker` (with WSL integration enabled if running in WSL)
- `kind`
- `kubectl`
- `helm`
- `openssl`
- project-local `go` toolchain available via `bin/` (already used by Makefile)

## Quick Start

```bash
hack/demo/setup-kind.sh
hack/demo/deploy-operator-kind.sh
hack/demo/apply-receivers.sh
hack/demo/apply-resourceactions.sh
hack/demo/trigger-actions.sh
hack/demo/show-logs.sh
```

To use a custom operator image tag:

```bash
IMG=resource-action-operator:dev hack/demo/deploy-operator-kind.sh
```

## Notes

- Namespace for demo resources: `ra-demo`
- Operator namespace: `resource-action-operator-system`
- Operator deploy method: Helm chart (`charts/resource-action-operator`)
- TLS uses a local demo CA and a service certificate for:
  - `ra-nginx-tls.ra-demo.svc`
  - `ra-nginx-tls.ra-demo.svc.cluster.local`

## Cleanup

```bash
hack/demo/cleanup.sh
```

## Reset / Recreate

Soft reset (keep cluster, reinstall everything):

```bash
hack/demo/reset.sh
```

Full reset (delete and recreate cluster):

```bash
FULL_RESET=true hack/demo/reset.sh
```
