# Resource Action Operator

![Kubernetes](https://img.shields.io/badge/kubernetes-operator-blue)
![Apache 2.0](https://img.shields.io/badge/license-Apache--2.0-green)
![Status](https://img.shields.io/badge/status-v0.1.0--alpha-orange)

**Repository:** [https://github.com/zdmr-space/resource-action-operator](https://github.com/zdmr-space/resource-action-operator)

- reacts to `Create`, `Update`, `Delete`
- filters by GVK, name/namespace regex, current labels, and label transitions on `Update`
- executes HTTP calls with retries, timeouts, and expected status matching
- creates Kubernetes Jobs for script/container-based actions
- persists richer Job execution details including job name, pod name, status, exit code, and optional log tail
- supports TLS/mTLS via Kubernetes secrets
- supports URL safety policy (allow/block regex, default local/metadata protection)
- stores execution history, retry/backoff metrics, and conditions in `status`
- emits Kubernetes Events for action success/failure summaries

## Overview

Resource Action Operator is a Kubernetes operator that allows you to execute **HTTP-based actions** in reaction to **Kubernetes resource lifecycle events** (Create / Update / Delete).

It is designed for:

* Platform teams
* Internal automation
* Webhook-style integrations
* GitOps-friendly event handling

Instead of embedding automation logic into applications, you declaratively describe **what should happen when a resource event occurs**.

---

## Core Concepts

### ResourceAction (CRD)

A `ResourceAction` defines:

* **Which resource** to observe (GVK selector)
* **Which events** to react to (Create / Delete)
* **Optional filters** (name, namespace, labels)
* **One or more actions** to execute

```yaml
apiVersion: ops.yusaozdemir.de/v1alpha1
kind: ResourceAction
metadata:
  name: deployment-webhook
  namespace: default
spec:
  selector:
    group: apps
    version: v1
    kind: Namespace
  events:
    - Create
  filters:
    nameRegex: "^demo-.*"
  actions:
    - type: http
      url: https://httpbin.org/post
      method: POST
```

HTTP authentication data should be referenced from Secrets, for example via `headers`.

## Actions

### HTTP Action

Currently supported action type:

* `http`

Features:

* Custom HTTP method
* Headers from Secrets
* JSON body templating
* Timeout
* Retry with backoff
* TLS / mTLS
* Expected status validation (regex)

---

### Body Templating

The request body supports Go templates.

Available fields:

```json
{
  "metadata": {
    "name": "<resource name>",
    "namespace": "<resource namespace>",
    "uid": "<resource uid>",
    "labels": { "key": "value" }
  }
}
```

Example:

```yaml
body:
  template: |
    {
      "event": "namespace-created",
      "name": "{{ .metadata.name }}",
      "uid": "{{ .metadata.uid }}"
    }
```

## Label-Based Matching

### expectedStatus (Optional)

You can define which HTTP status codes are considered **successful** using a regex.

```yaml
expectedStatus: "^2..$"   # default
expectedStatus: "^(4..|5..)$"
```

If the response status does not match:

* The action is treated as failed
* Retries may apply
* Status is updated accordingly

- `Deployment`
- `Service`
- `Namespace`
- `Node`

## Retry & Reliability

```yaml
retry:
  maxAttempts: 5
  backoff: 500ms
  maxBackoff: 10s
  retryOnStatus:
    - 429
    - 500
```

Supports:

* Exponential backoff with jitter
* Retry on network errors
* Retry on configurable HTTP status codes

---

## TLS & mTLS

```yaml
tls:
  insecureSkipVerify: false
  caSecretRef:
    name: ca-cert
    key: ca.pem
  clientCertSecretRef:
    name: client-cert
    certKey: tls.crt
    keyKey: tls.key
```

* Uses Kubernetes Secrets
* Namespaced isolation
* Supports full mTLS

---

## Action Modes

### once (default)

Executed **once per resource UID + event**.

Duplicate executions are prevented.

### cron

```yaml
mode: cron
schedule: 30s
```

* Periodic execution
* Independent of resource events
* Useful for heartbeats or sync jobs

Rules:

## Status & Conditions

Each ResourceAction maintains execution history and readiness state.

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ActionSucceeded
      message: All actions executed successfully
  executions:
    - event: Create
      resourceUID: <uid>
      executedAt: <timestamp>
```

* `Ready=True` → last execution successful
* `Ready=False` → last execution failed

---

## RBAC & Security Model

Security is **namespace-scoped by design**:

* ResourceActions are namespaced
* Secrets are read only from the same namespace
* No cross-namespace execution
* Operator only updates its own CR status

Recommended:

* One operator instance per cluster
* Use namespaces to isolate teams/projects

---

## Installation

### Local Development

```bash
make manifests
make run
```

Uses your current kubeconfig.

---

### Cluster Deployment

Apply CRD:

```bash
kubectl apply -f https://raw.githubusercontent.com/zdmr-space/resource-action-operator/v0.1.0/config/crd/bases/resourceactions.ops.yusaozdemir.de.yaml
```

Deploy controller (example):

```bash
kubectl apply -f config/manager/manager.yaml
```

Notes:
- `make deploy` works without cert-manager.
- `make deploy-webhook` requires cert-manager CRDs/controllers installed.

## Testing

- CRD/API: `api/v1alpha1`
- Controller: `internal/controller`
- Engine/Executor: `internal/engine`
- Manifests: `config/`
- Demo setup: `hack/demo/`
- ResourceAction templates: `template-files/demo/`
- Job action example: `template-files/resourceaction-job-bash.yaml`
- Generic ResourceAction chart: `charts/resource-action`
- Docs (AsciiDoc): `docs/`

kubectl describe resourceaction
```

AsciiDoc structure is prepared under `docs/`.

## Roadmap

Planned for future releases:

* Helm chart
* ValidatingAdmissionWebhook
* URL allowlists
* Metrics / Prometheus
* Multi-action chaining

URL policy details and examples:

## Contributing

Contributions are welcome!

1. Fork the repository
2. Create a feature branch
3. Add tests & documentation
4. Open a pull request

---

## License

Apache License 2.0

Additional Job metrics now include:

## Maintainer

**Batur Yusa Özdemir**
**Ahmet Taha Özdemir**
GitHub: [https://github.com/zdmr-space](https://github.com/zdmr-space)
