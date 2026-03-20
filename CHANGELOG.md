# Changelog

## 0.3.2

- added a central Antora-based demo documentation page under `docs/`
- documented end-to-end demo scenarios and their dependencies in a clearer order
- added GitHub Pages publishing for the AsciiDoc documentation via Antora
- added Helm values examples for operator Node watch RBAC and a Node-triggered Ubuntu Job demo

## 0.3.1

- cleaned up the repository structure and removed local demo and devcontainer scaffolding
- renamed `template-files/` to `examples/` and added focused example manifests
- refreshed the README and AsciiDoc documentation for Helm, quickstart, and chart values
- kept webhook, URL policy, and Helm-based installation as the documented default paths

## 0.2.0

- added `type: job` actions to create Kubernetes Jobs from `ResourceAction` objects
- added validation for job actions, service account usage, and action-specific fields
- documented HTTP/HTTPS usage, TLS options, job actions, and security recommendations
- added a job action example manifest
- aligned project version metadata for the `0.2.0` release
- added Helm image registry/repository/tag/digest configuration for operator deployment
- added a dedicated `resource-action-job` Helm chart for creating job-based ResourceActions
