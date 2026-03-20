# Examples

This directory contains focused example manifests for the Resource Action Operator.

For a guided walkthrough of the demo scenarios and the order in which supporting resources and `ResourceAction` objects should be applied, see:

- `docs/modules/ROOT/pages/demo.adoc`

The main examples are:

- `rbac/node-watcher-rbac.yaml`: additional RBAC so the operator can watch `Node` resources
- `node-label-update-http/resourceaction.yaml`: trigger an HTTP request when a `Node` label changes on update
- `job-ubuntu-secret/resourceaction.yaml`: trigger a Kubernetes Job using an Ubuntu image
- `job-ubuntu-secret/secret.example.yaml`: example Secret mounted by the Job example

## Apply Order

### Node label update with HTTP

1. Install the operator
2. Apply `rbac/node-watcher-rbac.yaml`
3. Apply `node-label-update-http/resourceaction.yaml`
4. Update a Node label so the transition matches

Example label change:

```bash
kubectl label node <node-name> demo.resource-action-operator/enabled=true --overwrite
```

### Ubuntu Job with mounted Secret

1. Install the operator
2. Apply `job-ubuntu-secret/secret.example.yaml`
3. Apply `job-ubuntu-secret/resourceaction.yaml`
4. Create a matching `Deployment`

## Notes

- Adjust namespaces, URLs, and image references before using these manifests outside a test cluster.
- For the HTTP example, replace the example URL with a reachable webhook endpoint.
- The `demo/` directory still contains larger end-to-end scenarios used for local testing.
