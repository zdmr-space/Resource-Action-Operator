# resource-action

Helm chart for creating a generic `ResourceAction`.

It supports:

- `type: http`
- `type: job`
- listener filters such as `labels`, `labelChanges`, `nameRegex`, and `namespaceRegex`

## Install

```bash
helm upgrade --install generic-action charts/resource-action \
  --namespace default
```

## HTTP example

```bash
helm upgrade --install generic-http charts/resource-action \
  --namespace default \
  --set action.type=http \
  --set selector.group=apps \
  --set selector.version=v1 \
  --set selector.kind=Deployment \
  --set events[0]=Create \
  --set http.url=https://example.internal/hook
```

## Job example

```bash
helm upgrade --install generic-job charts/resource-action \
  --namespace default \
  --set action.type=job \
  --set selector.group= \
  --set selector.version=v1 \
  --set selector.kind=Node \
  --set events[0]=Update \
  --set filters.labelChanges[0].key=demo.resource-action-operator/enabled \
  --set-string filters.labelChanges[0].to=true \
  --set job.allowRunAsRoot=true
```

## Published repository install

```bash
helm repo add resource-action-operator https://zdmr-space.github.io/Resource-Action-Operator
helm repo update

helm upgrade --install generic-action resource-action-operator/resource-action \
  --namespace default
```

## Example values file

```bash
helm upgrade --install generic-action charts/resource-action \
  --namespace default \
  -f charts/resource-action/values-example.yaml
```

## Notes

- `filters` is forwarded directly into the `ResourceAction` spec and supports `labels`, `labelChanges`, `nameRegex`, and `namespaceRegex`.
- For cluster-scoped resources such as `Node`, the operator ServiceAccount still needs the required watch RBAC.
- When `action.type=job`, the chart also supports `allowRunAsRoot`, `logTailLines`, `ttlSecondsAfterFinished`, volumes, mounts, env, and optional ServiceAccounts.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `nameOverride` | string | `""` | Short release name override. |
| `fullnameOverride` | string | `""` | Full release name override. |
| `resourceAction.create` | bool | `true` | Create the `ResourceAction`. |
| `resourceAction.name` | string | `""` | Override the generated `ResourceAction` name. |
| `resourceAction.namespace` | string | `""` | Namespace for the `ResourceAction`; defaults to the Helm release namespace. |
| `selector.group` | string | `"apps"` | Target resource API group. |
| `selector.version` | string | `"v1"` | Target resource API version. |
| `selector.kind` | string | `"Deployment"` | Target resource kind. |
| `events` | list | `["Create"]` | Trigger events for the `ResourceAction`. |
| `filters` | object | `{}` | Optional listener filters such as `labels`, `labelChanges`, `nameRegex`, or `namespaceRegex`. |
| `action.type` | string | `"http"` | Action type to render: `http` or `job`. |
| `action.mode` | string | `"once"` | Action mode, for example `once` or `cron`. |
| `action.schedule` | string | `""` | Duration string used when `action.mode` is `cron`. |
| `http.method` | string | `"POST"` | HTTP method for `type: http`. |
| `http.url` | string | `"https://example.internal/hook"` | Target URL for `type: http`. |
| `http.headers` | object | `{}` | Optional HTTP headers. |
| `http.body` | object | See `values.yaml` | Optional templated HTTP request body. |
| `http.expectedStatus` | string | `"^2..$"` | Regex used to validate the HTTP response code. |
| `http.timeout` | string | `"10s"` | HTTP action timeout. |
| `http.retry` | object | `{}` | Retry configuration for `type: http`. |
| `http.tls` | object | `{}` | TLS or mTLS configuration for HTTPS. |
| `http.urlPolicy` | object | `{}` | Optional URL policy overrides. |
| `job.image.registry` | string | `""` | Optional image registry prefix for the job runner image. |
| `job.image.repository` | string | `"bash"` | Job runner image repository. |
| `job.image.tag` | string | `"5.2"` | Job runner image tag, ignored when `job.image.digest` is set. |
| `job.image.digest` | string | `""` | Optional job runner image digest. |
| `job.interpreterCommand` | list | `["/bin/bash", "-c"]` | Interpreter command used for `job.script`. |
| `job.script` | string | `echo "deployment created"` | Inline script executed by the job image. |
| `job.command` | list | `[]` | Direct command array used instead of `job.script`. |
| `job.args` | list | `[]` | Arguments for `job.command`. |
| `job.env` | list | `[]` | Environment variables passed to the job. |
| `job.volumes` | list | `[]` | Secret or ConfigMap volumes exposed to the job. |
| `job.volumeMounts` | list | `[]` | Volume mounts for the defined volumes. |
| `job.serviceAccount.create` | bool | `false` | Create a dedicated ServiceAccount when `action.type=job`. |
| `job.serviceAccount.name` | string | `""` | ServiceAccount name to create or reference. |
| `job.serviceAccount.annotations` | object | `{}` | Annotations added to the created ServiceAccount. |
| `job.allowRunAsRoot` | bool | `false` | Allow images that require running as root by setting `runAsNonRoot=false` for the job container. |
| `job.automountServiceAccountToken` | bool | `false` | Mount the Kubernetes API token into the job Pod. |
| `job.timeout` | string | `"30s"` | Job timeout propagated to the `ResourceAction`. |
| `job.logTailLines` | int | `20` | Number of final job log lines to persist in `status.job.logTail`. |
| `job.ttlSecondsAfterFinished` | int | `300` | Job cleanup TTL after completion. |
| `job.backoffLimit` | int | `0` | Kubernetes Job retry limit. |
| `job.resources` | object | `{}` | CPU and memory requests/limits for the job container. |
