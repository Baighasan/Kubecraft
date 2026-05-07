# AGENTS.md

## Ownership Model

- **Terraform** owns infrastructure lifecycle (OCI network, compute, K3s host).
- **Helm** owns static control-plane Kubernetes resources (`charts/kubecraft-control-plane`).
- **Go code** owns dynamic tenant and server runtime resources (registration handler + CLI).

No operational path should apply raw Kubernetes manifests for control-plane or dynamic resources.

## Fast command map

- Dev CLI build (k3d defaults): `make build-dev`
- Prod CLI build (must override placeholders):
  `make build-prod PROD_ENDPOINT=<host:6443> PROD_NODE_ADDRESS=<public-ip>`
- Unit tests (subset only, no cluster needed): `make test`
- Integration tests (real cluster required): `go test -p 1 -tags=integration ./internal/...`
- Single package: `go test ./internal/cli/server`
- Single test: `go test ./internal/cli/server -run TestName`
- Local k3d cluster:
  1. `make cluster-up`
  2. `make cluster-setup`
  3. run tests / use CLI
  4. `make cluster-down`
- Helm control-plane validation: `helm lint ./charts/kubecraft-control-plane`
- Unit tests (subset only, no cluster needed): `make test`
- Integration tests (real cluster required): `go test -p 1 -tags=integration ./internal/...`
- Full test suite: `./scripts/test-all.sh` (Helm lint + Go integration tests)

## Project shape

- Single Go module: `github.com/baighasan/kubecraft`
- Two binaries:
  - `cmd/kubecraft` — Cobra CLI
  - `cmd/registration-server` — HTTP registration service
- `internal/k8s` — all Kubernetes orchestration (namespace/RBAC/server CRUD/scale/capacity/token)
- `internal/registration` — `/register` handler and username validation
- `internal/cli/server` — user-facing server commands (`create/list/start/stop/delete`)
- `charts/kubecraft-control-plane` — Helm chart for static control-plane resources (registration service + system RBAC)

## Authentication model

- No password. Registration returns a 5-year ServiceAccount token stored at `~/.kubecraft/config`.
- CLI commands (except `register`) require this config and initialize a namespaced K8s client at startup.
- Registration service runs in-cluster with a ClusterRole; users get namespace-scoped Roles only.

## Key constraints

- **Username/server name**: lowercase alnum only, length 3-16, must start with a letter.
- **NodePort range**: Minecraft servers use `30000-30015`; registration service is `30099`.
- **Capacity guard**: hard-coded for a single-node OCI model (14 Gi usable RAM). Creation is rejected if free RAM would drop below 4 Gi.
- **Server resources**: request 2 Gi / limit 4 Gi; 10 Gi PVC via `local-path` StorageClass.
- **Build-time ldflags**: `ClusterEndpoint`, `NodeAddress`, `TLSInsecure` are injected at build time (`Makefile`). Do not rely on runtime env vars for these.

## Gotchas

- `make test` is **not** `go test ./...`. It intentionally runs only non-integration packages (`internal/config`, `internal/registration`, `internal/cli`, `internal/cli/server`). `internal/k8s` tests are integration-only and require the `integration` build tag.
- Integration tests depend on `KUBECONFIG` (or `~/.kube/config`) and mutate cluster resources.
- `make cluster-setup` installs the control-plane Helm chart (`charts/kubecraft-control-plane`).
- Terraform variables are sensitive (OCID, SSH key, fingerprint) and excluded via `.gitignore` (`*.tfvars`, `.terraform/`).
