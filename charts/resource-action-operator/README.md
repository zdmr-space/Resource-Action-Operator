# resource-action-operator

Helm chart for deploying the Resource Action Operator.

## Install

```bash
helm upgrade --install resource-action-operator charts/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace
```

## Install from the published chart repository

```bash
helm repo add resource-action-operator https://zdmr-space.github.io/Resource-Action-Operator
helm repo update

helm upgrade --install resource-action-operator resource-action-operator/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace
```

## Operator image from a private registry

```bash
helm upgrade --install resource-action-operator charts/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace \
  --set image.registry=registry.example.com \
  --set image.repository=platform/resource-action-operator \
  --set image.tag=0.2.0-rc8
```

## Operator image by digest

```bash
helm upgrade --install resource-action-operator charts/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace \
  --set image.registry=registry.example.com \
  --set image.repository=platform/resource-action-operator \
  --set image.digest=sha256:<digest>
```

## Enable webhook with cert-manager

```bash
helm upgrade --install resource-action-operator charts/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace \
  --set webhook.enabled=true \
  --set webhook.certManager.enabled=true
```

## Optional extra RBAC for watched resources

The operator already gets the default permissions required for:

- `ResourceAction` resources
- creating `Job` objects
- emitting `Event` objects

If you want the operator to watch additional cluster-scoped resource types such as `Node`, add extra cluster rules:

```bash
helm upgrade --install resource-action-operator charts/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace \
  --set-json 'rbac.extraClusterRules=[{"apiGroups":[""],"resources":["nodes"],"verbs":["get","list","watch"]}]'
```

For extra namespaced permissions, use `rbac.extraRules`.

## Example values file

```bash
helm upgrade --install resource-action-operator charts/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace \
  -f charts/resource-action-operator/values-example.yaml
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `replicaCount` | int | `1` | Number of operator replicas. |
| `image.registry` | string | `""` | Optional image registry prefix for the operator image. |
| `image.repository` | string | `"controller"` | Operator image repository name. |
| `image.tag` | string | `"latest"` | Operator image tag, ignored when `image.digest` is set. |
| `image.digest` | string | `""` | Optional operator image digest. |
| `image.pullPolicy` | string | `"IfNotPresent"` | Kubernetes image pull policy. |
| `imagePullSecrets` | list | `[]` | Image pull secrets for private registries. |
| `nameOverride` | string | `""` | Short release name override. |
| `fullnameOverride` | string | `""` | Full release name override. |
| `serviceAccount.create` | bool | `true` | Create the operator ServiceAccount. |
| `serviceAccount.name` | string | `""` | Use an existing ServiceAccount name instead of the generated one. |
| `podAnnotations` | object | `{}` | Extra Pod annotations. |
| `podLabels` | object | `{}` | Extra Pod labels. |
| `resources` | object | See `values.yaml` | CPU and memory requests/limits for the operator Pod. |
| `nodeSelector` | object | `{}` | Node selector for the operator Pod. |
| `tolerations` | list | `[]` | Tolerations for the operator Pod. |
| `affinity` | object | `{}` | Affinity rules for the operator Pod. |
| `rbac.extraClusterRules` | list | `[]` | Additional ClusterRole rules, for example to watch cluster-scoped resources such as `nodes`. |
| `rbac.extraRules` | list | `[]` | Additional namespaced Role rules in the operator namespace. |
| `leaderElection` | bool | `true` | Enable controller-runtime leader election. |
| `healthProbeBindAddress` | string | `":8081"` | Health and readiness probe bind address. |
| `metrics.enabled` | bool | `true` | Enable the metrics endpoint. |
| `metrics.bindAddress` | string | `":8443"` | Metrics bind address passed to the manager. |
| `metrics.secure` | bool | `true` | Serve metrics over HTTPS. |
| `metrics.service.port` | int | `8443` | Kubernetes Service port for metrics. |
| `webhook.enabled` | bool | `false` | Enable admission webhook support. |
| `webhook.servicePort` | int | `443` | Service port exposed for the webhook. |
| `webhook.targetPort` | int | `9443` | Target port used by the webhook server in the Pod. |
| `webhook.path` | string | `"/validate-ops-yusaozdemir-de-v1alpha1-resourceaction"` | Admission webhook path. |
| `webhook.failurePolicy` | string | `"Fail"` | Admission webhook failure policy. |
| `webhook.caBundle` | string | `""` | Base64 PEM CA bundle when cert-manager is not used. |
| `webhook.certSecretName` | string | `"webhook-server-cert"` | Secret name containing the webhook serving cert. |
| `webhook.certMountPath` | string | `"/tmp/k8s-webhook-server/serving-certs"` | Mount path for serving certificates. |
| `webhook.certManager.enabled` | bool | `false` | Create cert-manager resources for webhook TLS. |
| `webhook.certManager.issuerName` | string | `"selfsigned-issuer"` | cert-manager Issuer name. |
| `webhook.certManager.certificateName` | string | `"serving-cert"` | cert-manager Certificate name. |
