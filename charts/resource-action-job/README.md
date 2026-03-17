# resource-action-job

Helm chart for creating a `ResourceAction` with `type: job`.

## Install

```bash
helm upgrade --install deployment-job charts/resource-action-job \
  --namespace default
```

## Install from the published chart repository

```bash
helm repo add resource-action-operator https://zdmr-space.github.io/Resource-Action-Operator
helm repo update

helm upgrade --install deployment-job resource-action-operator/resource-action-job \
  --namespace default
```

## Image by tag

```bash
helm upgrade --install deployment-job charts/resource-action-job \
  --namespace default \
  --set selector.group=apps \
  --set selector.version=v1 \
  --set selector.kind=Deployment \
  --set job.image.registry=registry.example.com \
  --set job.image.repository=platform/bash-runner \
  --set job.image.tag=1.2.3
```

## Image by digest

```bash
helm upgrade --install deployment-job charts/resource-action-job \
  --namespace default \
  --set job.image.registry=registry.example.com \
  --set job.image.repository=platform/bash-runner \
  --set job.image.digest=sha256:<digest>
```

## Dedicated ServiceAccount

```bash
helm upgrade --install deployment-job charts/resource-action-job \
  --namespace default \
  --set job.serviceAccount.create=true \
  --set job.serviceAccount.name=restricted-runner
```

## Secret and ConfigMap mounts

```bash
helm upgrade --install deployment-job charts/resource-action-job \
  --namespace default \
  --set job.volumes[0].name=tls \
  --set job.volumes[0].secret.secretName=api-client-cert \
  --set job.volumes[1].name=scripts \
  --set job.volumes[1].configMap.name=job-scripts \
  --set job.volumeMounts[0].name=tls \
  --set job.volumeMounts[0].mountPath=/var/run/tls \
  --set job.volumeMounts[1].name=scripts \
  --set job.volumeMounts[1].mountPath=/opt/scripts
```

## Example values file

```bash
helm upgrade --install deployment-job charts/resource-action-job \
  --namespace default \
  -f charts/resource-action-job/values-example.yaml
```

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
| `filters` | object | `{}` | Optional selector filters such as labels or regexes. |
| `job.mode` | string | `"once"` | Job action mode, typically `once` or `cron`. |
| `job.schedule` | string | `""` | Duration string used when `job.mode` is `cron`. |
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
| `job.serviceAccount.create` | bool | `false` | Create a dedicated ServiceAccount for the job action. |
| `job.serviceAccount.name` | string | `""` | ServiceAccount name to create or reference. |
| `job.serviceAccount.annotations` | object | `{}` | Annotations added to the created ServiceAccount. |
| `job.automountServiceAccountToken` | bool | `false` | Mount the Kubernetes API token into the job Pod. |
| `job.timeout` | string | `"30s"` | Job timeout propagated to the `ResourceAction`. |
| `job.ttlSecondsAfterFinished` | int | `300` | Job cleanup TTL after completion. |
| `job.backoffLimit` | int | `0` | Kubernetes Job retry limit. |
| `job.resources` | object | `{}` | CPU and memory requests/limits for the job container. |
