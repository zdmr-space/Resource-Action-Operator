# Resource Action Operator

A Kubernetes operator for event-driven HTTP and Job actions on Kubernetes resources.

## Features

- reacts to `Create`, `Update`, `Delete`
- filters by GVK, name/namespace regex, and labels
- executes HTTP calls with retries, timeouts, and expected status matching
- creates Kubernetes Jobs for script/container-based actions
- supports TLS/mTLS via Kubernetes secrets
- supports URL safety policy (allow/block regex, default local/metadata protection)
- stores execution history, retry/backoff metrics, and conditions in `status`
- emits Kubernetes Events for action success/failure summaries

## Requirements

- Go `1.24+`
- Docker
- kubectl
- Helm `3.15+`
- kind (for local clusters)

## Quickstart (local kind)

```bash
hack/demo/setup-kind.sh
hack/demo/deploy-operator-kind.sh
hack/demo/apply-receivers.sh
hack/demo/apply-resourceactions.sh
hack/demo/trigger-actions.sh
hack/demo/show-logs.sh
```

All-in-one run:

```bash
hack/demo/run-all.sh
```

## Reset / Cleanup

Soft cleanup (cluster stays):

```bash
hack/demo/cleanup.sh
```

Hard cleanup (cluster gets deleted):

```bash
DELETE_CLUSTER=true hack/demo/cleanup.sh
```

Soft reset (redeploy, keep cluster):

```bash
hack/demo/reset.sh
```

Hard reset (delete and recreate cluster):

```bash
FULL_RESET=true hack/demo/reset.sh
```

## Install with Helm (recommended)

Install/upgrade (works without cert-manager):

```bash
helm upgrade --install resource-action-operator charts/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace
```

Enable admission webhook with cert-manager:

```bash
helm upgrade --install resource-action-operator charts/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace \
  --set webhook.enabled=true \
  --set webhook.certManager.enabled=true
```

Uninstall:

```bash
helm uninstall resource-action-operator -n resource-action-operator-system
```

Install from the published chart repository:

```bash
helm repo add resource-action-operator https://zdmr-space.github.io/Resource-Action-Operator
helm repo update

helm upgrade --install resource-action-operator resource-action-operator/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace
```

Makefile helpers:

```bash
make helm-lint
make helm-template
make helm-install
make helm-install-webhook
make helm-uninstall
```

Use a custom registry/repository for the operator image:

```bash
helm upgrade --install resource-action-operator charts/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace \
  --set image.registry=registry.example.com \
  --set image.repository=platform/resource-action-operator \
  --set image.tag=0.2.0-rc5
```

Deploy a `ResourceAction` job via the dedicated Helm chart:

```bash
helm upgrade --install deployment-job charts/resource-action-job \
  --namespace default \
  --set selector.group=apps \
  --set selector.version=v1 \
  --set selector.kind=Deployment \
  --set job.image.registry=registry.example.com \
  --set job.image.repository=platform/bash-runner \
  --set job.image.tag=1.0.0
```

## Release Pipeline

GitHub Actions now includes:

- CI for Helm linting and rendering in [helm.yml](/c:/Entwicklung/resource-action-operator/.github/workflows/helm.yml)
- tag-based release publishing in [release.yml](/c:/Entwicklung/resource-action-operator/.github/workflows/release.yml)

Release workflow outputs:

- multi-arch operator image to `ghcr.io/zdmr-space/resource-action-operator`
- published Helm chart repository via GitHub Pages
- packaged releases for `resource-action-operator` and `resource-action-job`

Create a release by pushing a tag such as `v0.2.0-rc5` or `v0.2.0`.

For a manual test publish from a branch, run the `Release` workflow via `workflow_dispatch` and provide:

- `version`, for example `0.2.0-rc5`
- `publish_image=true`
- `publish_charts=true`

## Real Cluster Test

Recommended order on a real cluster:

1. Push the repository changes and create a tag such as `v0.2.0-rc5`.
2. Wait for the `Release` workflow to publish the image and charts.
3. Install the operator chart from the published repository.
4. Verify the operator Pod is running:

```bash
kubectl -n resource-action-operator-system get pods
```

5. Install a test job action:

```bash
helm upgrade --install deployment-job resource-action-operator/resource-action-job \
  --namespace default
```

6. Trigger the selected Kubernetes event and inspect:

```bash
kubectl get resourceaction -A
kubectl get jobs -A
kubectl logs -n resource-action-operator-system deploy/resource-action-operator-controller-manager
```

## Action Types

### `type: http`

Use `type: http` for HTTP or HTTPS requests. HTTPS is selected by using a `https://` URL.

Example with HTTPS and disabled certificate verification:

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
      tls:
        insecureSkipVerify: true
```

HTTP authentication data should be referenced from Secrets, for example via `headers`.

### `type: job`

Use `type: job` to create a Kubernetes Job that runs a script or command in a user-provided container image.

Job actions support:

- environment variables
- Secret-backed environment variables
- read-only Secret and ConfigMap volume mounts
- explicit ServiceAccount selection
- container resource limits

Example:

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
        volumes:
          - name: tls
            secret:
              secretName: api-client-cert
          - name: scripts
            configMap:
              name: job-scripts
        volumeMounts:
          - name: tls
            mountPath: /var/run/tls
          - name: scripts
            mountPath: /opt/scripts
        serviceAccountName: restricted-runner
        automountServiceAccountToken: false
        timeout: 30s
```

## Label-Based Matching

Matching on the current label set of a resource is already supported through `filters.labels`.

This works for namespaced and cluster-scoped resources, for example:

- `Deployment`
- `Service`
- `Namespace`
- `Node`

Example for a `Node` that should trigger on `Update` when it currently has a specific label:

```yaml
apiVersion: ops.yusaozdemir.de/v1alpha1
kind: ResourceAction
metadata:
  name: node-label-http-hook
  namespace: default
spec:
  selector:
    group: ""
    version: v1
    kind: Node
  events:
    - Update
  filters:
    labels:
      demo.resource-action-operator/enabled: "true"
  actions:
    - type: http
      method: POST
      url: https://example.internal/hook
```

For cluster-scoped resources such as `Node`, the operator ServiceAccount needs additional RBAC permissions to watch that resource type.

## Security Model

- Job actions run as Kubernetes Jobs, not as local processes inside the operator container.
- Job Pods default to `automountServiceAccountToken=false`.
- Job Pods use restrictive container defaults such as `allowPrivilegeEscalation=false` and dropped Linux capabilities.
- Additional Kubernetes API permissions should be granted only through explicitly created ServiceAccounts and RoleBindings.
- HTTP auth tokens, API keys, and certificates should be stored in Kubernetes Secrets and referenced from the `ResourceAction`.

## Roadmap

Planned for a follow-up release:

- label-change based triggers for `Update` events, for example when a resource receives a specific label and this transition should trigger an HTTP action or Job action
- richer update-aware filters that can evaluate old versus new label state instead of only matching the current label set

## Development

```bash
make test
make run
make build
```

Kustomize-based deploy to current kube context:

```bash
make deploy IMG=<image>
```

Deploy with admission webhook + cert-manager wiring:

```bash
make deploy-webhook IMG=<image>
```

Notes:
- `make deploy` works without cert-manager.
- `make deploy-webhook` requires cert-manager CRDs/controllers installed.

## Important Paths

- CRD/API: `api/v1alpha1`
- Controller: `internal/controller`
- Engine/Executor: `internal/engine`
- Manifests: `config/`
- Demo setup: `hack/demo/`
- ResourceAction templates: `template-files/demo/`
- Job action example: `template-files/resourceaction-job-bash.yaml`
- Docs (AsciiDoc): `docs/`

## Documentation

AsciiDoc structure is prepared under `docs/`.

Entry point:

```bash
docs/README.adoc
```

URL policy details and examples:

```bash
docs/modules/ROOT/pages/url-policy.adoc
```

## Metrics

The operator exposes Prometheus metrics on the controller-runtime metrics endpoint, including:

- `resource_action_operator_http_runs_total{result}`
- `resource_action_operator_http_actions_total`
- `resource_action_operator_http_attempts_total`
- `resource_action_operator_http_retries_total{type}`
- `resource_action_operator_http_backoff_seconds_total`
- `resource_action_operator_http_duration_seconds{result}`
- `resource_action_operator_http_last_status_total{class}`

Detailed usage and PromQL examples:

```bash
docs/modules/ROOT/pages/metrics.adoc
```

## License

Apache-2.0
