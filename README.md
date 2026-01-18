# Resource Action Operator

A Kubernetes operator that executes HTTP actions in response to Kubernetes resource events.

The **Resource Action Operator** allows platform teams to define declarative, event-driven automations using Custom Resources. It is designed to be **secure**, **namespaced**, and **multi-tenant friendly**.

---

## ✨ Features

* Reacts to Kubernetes resource **Create / Update / Delete** events
* Declarative **HTTP actions** (webhooks, APIs, automation endpoints)
* Namespaced `ResourceAction` CRD (safe for multi-tenant clusters)
* Regex-based **expectedStatus** validation (e.g. `^2..$`, `^(4..|5..)$`)
* Retries with backoff, timeouts, TLS & mTLS support
* Cron-based periodic actions
* Execution history & Conditions stored in status
* RBAC-isolated by namespace

---

## 🧠 Core Concepts

### ResourceAction

A `ResourceAction` defines:

* **Which Kubernetes resources** to watch (GVK selector)
* **Which events** trigger actions (Create/Delete)
* **Optional filters** (name regex, labels)
* **One or more actions** (HTTP calls)

Each `ResourceAction` is **namespaced**.

---

## 📦 Custom Resource Definition

```yaml
apiVersion: ops.yusaozdemir.de/v1alpha1
kind: ResourceAction
metadata:
  name: namespace-create-once
  namespace: default
spec:
  selector:
    group: ""
    version: v1
    kind: Namespace

  events:
    - Create

  filters:
    nameRegex: "^demo-.*"

  actions:
    - type: http
      mode: once
      method: POST
      url: https://httpbin.org/status/404
      expectedStatus: "^(4..|5..)$"
      timeout: 10s
      body:
        template: |
          {
            "event": "namespace-created",
            "name": "{{ .metadata.name }}",
            "uid": "{{ .metadata.uid }}"
          }
```

---

## 🔁 Action Modes

### once

Executed once per resource UID and event.

### cron

Executed periodically using a Go duration (e.g. `30s`, `5m`).

```yaml
mode: cron
schedule: 30s
```

---

## 🌐 HTTP Action Options

| Field          | Description                                 |
| -------------- | ------------------------------------------- |
| method         | HTTP method (default: POST)                 |
| url            | Target endpoint                             |
| headers        | Static or secret-based headers              |
| body.template  | Go template rendered from resource metadata |
| timeout        | Request timeout                             |
| expectedStatus | Regex matching HTTP status code             |
| retry          | Retry policy                                |
| tls            | TLS / mTLS configuration                    |

---

## 🔐 Security Model

### Namespaced Isolation

* `ResourceAction` is namespaced
* Teams can only create actions in their namespace
* No cross-namespace execution

### RBAC

**Team Role Example:**

```yaml
kind: Role
rules:
- apiGroups: ["ops.yusaozdemir.de"]
  resources: ["resourceactions"]
  verbs: ["get","list","watch","create","update","delete"]
```

The operator itself runs with a ClusterRole that allows watching resources and updating status.

---

## 📊 Status & Conditions

Each execution updates status:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ActionSucceeded
      message: All actions executed successfully
      lastTransitionTime: "2026-01-18T14:08:02Z"

  executions:
    - event: Create
      resourceUID: 075e834a-64f3-4401-a958-67511d5bd4b7
      executedAt: "2026-01-18T14:08:02Z"
```

---

## 🚀 Deployment

### Prerequisites

* Kubernetes >= 1.27
* kubectl
* Go 1.22+

### Install CRD

```bash
kubectl apply -f config/crd/bases/resourceactions.ops.yusaozdemir.de.yaml
```

### Run Locally (Development)

```bash
make run
```

### Deploy to Cluster

```bash
make docker-build docker-push IMG=<your-image>
make deploy IMG=<your-image>
```

---

## 🧪 Testing

```bash
kubectl create namespace demo-test
kubectl delete namespace demo-test

kubectl describe resourceaction namespace-create-once
```

---

## 🧭 Roadmap

* ValidatingWebhook (URL allowlists)
* Metrics (Prometheus)
* Helm chart
* Action chaining
* Audit logging

---

## 📜 License

Apache License 2.0

---

## ❤️ Contributing

Contributions are welcome!

* Fork the repo
* Create a feature branch
* Add tests if applicable
* Open a Pull Request

---

## ✉️ Maintainer

Created by **Batur Yusa Özdemir** and **Ahmet Taha Özdemir**

