# Legacy Manifest Templates (Reference Only)

The YAML files in this directory are preserved for reference only.

**Do not apply these manifests operationally.**

- `user-templates/` — correspond to namespace-scoped resources now created dynamically by the registration service (`internal/registration/handler.go`) and the Go K8s client (`internal/k8s/`).
- `server-templates/` — correspond to per-server workload resources now created dynamically by the CLI server commands (`internal/cli/server/`).

Dynamic resources must be created exclusively through the Go runtime code paths so that naming conventions, resource limits, port allocation, and RBAC bindings remain consistent with the in-code logic.

Static control-plane resources (namespace, registration service deployment, cluster-scoped RBAC) are owned by the Helm chart at `charts/kubecraft-control-plane/`.
