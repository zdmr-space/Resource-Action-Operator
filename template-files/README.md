# Template Files

This directory contains example manifests for the Resource Action Operator.

The files are split into two groups:

- Top-level YAML files are small standalone examples that show common `ResourceAction` patterns.
- The `demo/` directory contains multi-file demo scenarios that are intended for end-to-end testing on a cluster.

These files are meant as starting points. Adjust namespaces, URLs, filters, images, service accounts, and security settings before using them in production.

## Prerequisites

Before testing any template in this directory, make sure the following components are available on your cluster:

1. A working Kubernetes cluster
2. `kubectl` configured for that cluster
3. `helm` installed locally
4. The Resource Action Operator installed on the cluster

If you want to use the published Helm charts, add the chart repository first:

```bash
helm repo add resource-action-operator https://zdmr-space.github.io/Resource-Action-Operator
helm repo update
```

Then install the operator:

```bash
helm upgrade --install resource-action-operator resource-action-operator/resource-action-operator \
  --namespace resource-action-operator-system \
  --create-namespace
```

Verify that the operator is running before applying any example:

```bash
kubectl get pods -n resource-action-operator-system
kubectl logs -n resource-action-operator-system deploy/resource-action-operator-controller-manager
```

For `job`-based examples, also make sure that:

- the job image can be pulled by your cluster
- the referenced service account exists if the example uses one
- any referenced `Secret` or `ConfigMap` already exists

## Test Order

The general test order is always the same:

1. Install the operator
2. Apply the `ResourceAction`
3. Apply or create the resource that should trigger the action
4. Inspect the generated Jobs, logs, or HTTP target

For multi-file demos, create supporting resources first, then the `ResourceAction`, and only then the trigger resource.

## Top-Level Examples

### `namespace-create-once.yaml`

Namespace-focused HTTP examples with filters.

It contains three `ResourceAction` objects:

- A `Create` action that reacts once for namespaces whose name matches `^demo-.*`
- A `Create` action in `cron` mode that repeatedly calls an HTTP endpoint
- A `Delete` action that reacts once when a matching namespace is removed

This file is useful when you want to understand:

- event selection
- name-based filtering
- `once` versus `cron`
- templated HTTP request bodies
- expected status code matching

### `resourceaction-default.yaml`

A minimal HTTP example for namespace create and delete events.

It demonstrates:

- selecting core Kubernetes resources such as `Namespace`
- sending a simple HTTP POST
- sending a templated JSON payload
- running more than one action in a single `ResourceAction`

Use this as the simplest starting point for webhook-style automation.

### `resourceaction-cron.yaml`

A compact example for scheduled follow-up actions.

It shows:

- a `Create` event that starts a recurring HTTP action with `mode: cron`
- a `Delete` event that still uses `mode: once`

Use this when a resource event should start a recurring callback instead of a one-time notification.

### `resourceaction-job-bash.yaml`

A basic `job` action example for `Deployment` creation.

It demonstrates:

- `type: job`
- running a Bash script inside a Kubernetes Job
- mounting a `ConfigMap` as a volume
- setting a dedicated `serviceAccountName`
- disabling automatic service account token mounting
- configuring timeout and TTL cleanup

Use this as the starting point for script-based job execution.

## Demo Scenarios

The `demo/` directory contains more realistic examples that can be applied to a cluster as a small test setup.

### `demo/namespace-http-sink.yaml`

Creates a dedicated namespace, an NGINX deployment, and a service that acts as a simple in-cluster HTTP sink.

Use it when you want to verify that HTTP actions from the operator actually reach a live service.

### `demo/resourceaction-namespace-http.yaml`

Creates a `ResourceAction` that reacts to namespace creation events whose names match a configured prefix and sends an HTTP request to the in-cluster NGINX sink.

This is useful for validating:

- namespace event watches
- regex-based filtering
- in-cluster HTTP calls

### `demo/trigger-namespace-http.yaml`

Creates a namespace that is meant to match the HTTP demo filter and trigger the HTTP action.

Apply this after the HTTP sink and the matching `ResourceAction` are already in place.

### `demo/resourceaction-namespace-job-bash.yaml`

Creates a `ResourceAction` that reacts to namespace creation events with a matching prefix and launches a Kubernetes Job that runs a small Bash script.

Use it to validate the `job` action path without needing external systems.

### `demo/trigger-namespace-job.yaml`

Creates a namespace that is meant to match the Bash job demo filter and trigger the job execution.

Apply this after the matching job-based `ResourceAction` is installed.

### `demo/resourceaction-deployment-http.yaml`

An example `ResourceAction` for reacting to `Deployment` events and sending an HTTP request.

Use it when you want to test application-level resources instead of namespace events.

### `demo/resourceaction-deployment-tls.yaml`

A TLS-oriented HTTP example for `Deployment` events.

Use it as a reference for HTTPS-related configuration such as custom CA handling or relaxed verification during testing.

### `demo/operator-node-watch-rbac.yaml`

Grants the operator the extra cluster-scoped permissions needed for Node-based demos.

Apply this before testing `Node`-triggered actions.

### `demo/resourceaction-node-label-http.yaml`

Creates a `ResourceAction` that reacts to `Node` update events and matches Nodes that currently have the label `demo.resource-action-operator/enabled=true`.

It sends an HTTP request to the in-cluster NGINX sink.

### `demo/resourceaction-node-label-job.yaml`

Creates a `ResourceAction` that reacts to `Node` update events and matches Nodes that currently have the label `demo.resource-action-operator/enabled=true`.

It launches a Kubernetes Job and is useful for testing label-based Job execution on cluster-scoped resources.

### `demo/resourceaction-node-label-transition-job.yaml`

Creates a `ResourceAction` that reacts only when a `Node` label changes on update, for example when `demo.resource-action-operator/enabled` transitions from absent to `true`.

Use it to validate old-versus-new label matching instead of only matching the current label set.

## Typical Usage

Apply a simple standalone example:

```bash
kubectl apply -f template-files/resourceaction-default.yaml
```

Then create or update a matching resource to trigger it. For example:

```bash
kubectl create namespace demo-test
```

Run the namespace-to-HTTP demo in the correct order:

```bash
kubectl apply -f template-files/demo/namespace-http-sink.yaml
kubectl apply -f template-files/demo/resourceaction-namespace-http.yaml
kubectl apply -f template-files/demo/trigger-namespace-http.yaml
```

Run the namespace-to-job demo in the correct order:

```bash
kubectl apply -f template-files/demo/resourceaction-namespace-job-bash.yaml
kubectl apply -f template-files/demo/trigger-namespace-job.yaml
```

Run the node-label-to-HTTP demo in the correct order:

```bash
kubectl apply -f template-files/demo/namespace-http-sink.yaml
kubectl apply -f template-files/demo/operator-node-watch-rbac.yaml
kubectl apply -f template-files/demo/resourceaction-node-label-http.yaml
kubectl label node <node-name> demo.resource-action-operator/enabled=true --overwrite
```

Run the node-label-to-job demo in the correct order:

```bash
kubectl apply -f template-files/demo/operator-node-watch-rbac.yaml
kubectl apply -f template-files/demo/resourceaction-node-label-job.yaml
kubectl label node <node-name> demo.resource-action-operator/enabled=true --overwrite
```

Run the node label transition demo in the correct order:

```bash
kubectl apply -f template-files/demo/operator-node-watch-rbac.yaml
kubectl apply -f template-files/demo/resourceaction-node-label-transition-job.yaml
kubectl label node <node-name> demo.resource-action-operator/enabled=true --overwrite
```

Inspect the result:

```bash
kubectl get resourceactions -A
kubectl get jobs -A
kubectl logs -n resource-action-operator-system deploy/resource-action-operator-controller-manager
```

For the HTTP sink demo, you can also inspect the sink workload:

```bash
kubectl get pods -n demo-http-sink
kubectl logs -n demo-http-sink deploy/ra-http-sink
```

## Notes

- Some examples use public test endpoints such as `httpbin.org`. Replace them for internal or production use.
- Job examples may require an existing service account such as `restricted-runner`.
- Demo manifests are intentionally simple and optimized for validation, not for hardened production deployment.
- For `Node`-based demos, make sure the operator has cluster-scoped watch permissions for `nodes`.
