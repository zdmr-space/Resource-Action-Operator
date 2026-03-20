# Resource Action Operator

Kubernetes operator for reacting to resource events and executing follow-up actions.

The operator watches Kubernetes resources such as `Deployment`, `Namespace`, or `Node` and triggers either:

- HTTP requests for webhook and API integrations
- Kubernetes Jobs for script- or container-based automation

## What It Does

- reacts to `Create`, `Update`, and `Delete`
- matches resources by GVK plus optional name, namespace, label, and label-transition filters
- executes HTTP actions with timeout, retries, TLS/mTLS, and expected status validation
- executes Job actions with user-provided images, scripts, env vars, mounts, and service accounts
- stores execution state, conditions, and failure details in `status`
- emits Kubernetes Events for successful and failed runs

## Typical Use Cases

- call a webhook when a `Deployment` is created
- trigger a Job when a label is added to a `Node`
- start recurring follow-up callbacks after a matching event
- run cluster-local automation without embedding custom logic into applications

## Requirements

- Kubernetes cluster
- Helm `3.15+` recommended for installation
- `kubectl`

For local development and demo runs, this repository also uses Go, Docker, and kind.

## Install

Install from the local chart:

```bash
helm upgrade --install resource-action-operator charts/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace
```

Install from the published chart repository:

```bash
helm repo add resource-action-operator https://zdmr-space.github.io/Resource-Action-Operator
helm repo update

helm upgrade --install resource-action-operator resource-action-operator/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace
```

Enable the admission webhook with cert-manager:

```bash
helm upgrade --install resource-action-operator charts/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace \
  --set webhook.enabled=true \
  --set webhook.certManager.enabled=true
```

## First Example

Apply a minimal HTTP-based `ResourceAction`:

```bash
kubectl apply -f examples/resourceaction-default.yaml
```

Then create a matching resource, for example:

```bash
kubectl create namespace demo-test
```

For more manifest examples, see `examples/`. For a guided walkthrough of the demo scenarios, see `docs/modules/ROOT/pages/demo.adoc`. For action details, see `docs/modules/ROOT/pages/actions.adoc`. For Helm installation and chart values, see `docs/modules/ROOT/pages/helm.adoc`. For reusable Helm-based actions, see `charts/resource-action` and `charts/resource-action-job`.

## Action Types

### `type: http`

Use HTTP actions for webhooks and API calls.

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
    kind: Deployment
  events:
    - Create
  actions:
    - type: http
      method: POST
      url: https://example.internal/hook
      expectedStatus: "^2..$"
```

### `type: job`

Use Job actions to create a Kubernetes Job from a user-defined image and script.

```yaml
apiVersion: ops.yusaozdemir.de/v1alpha1
kind: ResourceAction
metadata:
  name: deployment-job
  namespace: default
spec:
  selector:
    group: apps
    version: v1
    kind: Deployment
  events:
    - Create
  actions:
    - type: job
      mode: once
      job:
        image: bash:5.2
        interpreterCommand:
          - /bin/bash
          - -c
        script: |
          echo "deployment created"
        timeout: 30s
```

Job actions support:

- scripts or direct commands
- environment variables and Secret-backed environment variables
- Secret and ConfigMap mounts
- explicit `serviceAccountName`
- resource limits
- cleanup via `ttlSecondsAfterFinished`

## Matching and Filters

The operator selects resources by:

- API group, version, and kind
- event type: `Create`, `Update`, `Delete`
- optional `nameRegex`
- optional `namespaceRegex`
- optional `filters.labels`
- optional `filters.labelChanges` for update transitions

Example for matching a `Node` update when a label changes to `true`:

```yaml
filters:
  labelChanges:
    - key: demo.resource-action-operator/enabled
      to: "true"
```

Cluster-scoped resources such as `Node` require the operator to have watch permissions for that resource type.

## Security Notes

- treat `ResourceAction` write access as privileged
- keep Job actions on restricted service accounts
- store tokens, certificates, and secrets in Kubernetes `Secret` objects
- grant additional watch permissions explicitly through RBAC

## Charts

This repository contains three Helm charts:

- `charts/resource-action-operator`: installs the operator
- `charts/resource-action`: creates a generic `ResourceAction`
- `charts/resource-action-job`: creates a job-focused `ResourceAction`

## Repository Layout

- `api/v1alpha1`: CRD types, validation, webhook logic
- `internal/controller`: controller-runtime reconciliation and watches
- `internal/engine`: execution logic for HTTP and Job actions
- `charts/`: Helm charts for operator and reusable actions
- `config/`: generated and deployment manifests
- `examples/`: standalone example manifests and demo scenarios
- `docs/`: AsciiDoc documentation
- `test/`: end-to-end test helpers and suites

## Development

Run tests:

```bash
make test
```

Run the operator locally against the current kube context:

```bash
make run
```

Deploy with Kustomize:

```bash
make deploy IMG=<image>
```

Deploy with webhook support:

```bash
make deploy-webhook IMG=<image>
```

## Documentation

- docs entry point: `docs/README.adoc`
- quickstart: `docs/modules/ROOT/pages/quickstart.adoc`
- demo scenarios: `docs/modules/ROOT/pages/demo.adoc`
- action details: `docs/modules/ROOT/pages/actions.adoc`
- Helm usage: `docs/modules/ROOT/pages/helm.adoc`

## License

Apache-2.0
