# AGENTS.md

## Fast command map

- Dev CLI build (k3d defaults): `make build-dev`
- Prod CLI build (must override placeholders):
  `make build-prod PROD_ENDPOINT=<host:6443> PROD_NODE_ADDRESS=<public-ip>`
- Unit tests (subset only, no cluster needed): `make test`
- Integration tests (real cluster required): `go test -tags=integration ./internal/...`
- Single package: `go test ./internal/cli/server`
- Single test: `go test ./internal/cli/server -run TestName`
- Local k3d cluster:
  1. `make cluster-up`
  2. `make cluster-setup`
  3. run tests / use CLI
  4. `make cluster-down`
- Manifest validation: `./scripts/test-manifests.sh`
- RBAC functional tests: `./scripts/test-rbac.sh`
- Full test suite: `./scripts/test-all.sh`

## Project shape

- Single Go module: `github.com/baighasan/kubecraft`
- Two binaries:
  - `cmd/kubecraft` — Cobra CLI
  - `cmd/registration-server` — HTTP registration service
- `internal/k8s` — all Kubernetes orchestration (namespace/RBAC/server CRUD/scale/capacity/token)
- `internal/registration` — `/register` handler and username validation
- `internal/cli/server` — user-facing server commands (`create/list/start/stop/delete`)
- `manifests/*-templates` — YAML templates with `{username}` and `{servername}` placeholders

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
- `make cluster-setup` applies `registration-namespace.yaml` **before** the rest of the registration templates to avoid ordering issues when applying the whole directory.
- There is a naming mismatch between code and manifests for the capacity-checker ClusterRole:
  - Code constant: `kubecraft-capacity-checker`
  - Manifest file (`system-templates/clusterrole.yaml`): `kc-capacity-checker`
  Keep names consistent when modifying RBAC; apply the manifest that matches your target.
- Terraform variables are sensitive (OCID, SSH key, fingerprint) and excluded via `.gitignore` (`*.tfvars`, `.terraform/`).
