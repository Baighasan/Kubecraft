# Kubecraft Agent Notes

## What this repo is
- Single Go module CLI + in-cluster registration service; entrypoints are `cmd/kubecraft/main.go` and `cmd/registration-server/main.go`.
- Static control-plane resources are Helm-managed in `charts/kubecraft-control-plane`; dynamic tenant/server resources are created by Go code in `internal/k8s` + `internal/registration`.

## Commands you should not guess
- Build CLI with image ldflags: `make build` (injects `internal/config.ServerImage` with `SERVER_IMAGE_TAG`, default `dev`).
- Unit test scope used by Makefile/CI: `make test` (targets `internal/config`, `internal/registration`, `internal/cli`, `internal/cli/server`).
- Integration tests require a real cluster and run serially: `go test -v -race -p 1 -tags=integration ./internal/...`.
- Helm chart checks used in CI: `helm lint ./charts/kubecraft-control-plane` and `helm template kubecraft-control-plane ./charts/kubecraft-control-plane | kubectl apply --dry-run=client -f -`.
- End-to-end local validation script: `./scripts/test-all.sh` (Helm validate -> Helm install -> integration tests).

## Test and environment gotchas
- Integration tests assume Kubernetes access via `KUBECONFIG` (or default kubeconfig) and expect control-plane RBAC/chart resources to exist.
- Keep `-p 1` on integration runs; tests mutate shared cluster-scoped RBAC (`kc-users-capacity-check`) and can conflict in parallel.
- `kubecraft init --ip` accepts only literal IPs (not DNS names), despite README wording.

## Runtime/config facts that affect edits
- User config is persisted to `~/.kubecraft/config` with fields `clusterIP`, `tlsInsecure`, `username`, `token`.
- `init` probes Kubernetes API on `https://<clusterIP>:6443` and may persist `tlsInsecure: true` after TLS verification fallback.
- Registration endpoint is `http://<clusterIP>:30099/register`; in-cluster registration service itself listens on `:8080`.
- Server image override precedence in `server create`: `--server-image` flag, else `KUBECRAFT_SERVER_IMAGE`, else build-time default from `internal/config.ServerImage`.

## CI/release conventions to preserve
- PRs to `main` run unit/integration/manifests workflows based on path filters in `.github/workflows/`.
- Release automation only runs when a PR is merged to `main` with `release` label; version bump is controlled by optional `semver:major` / `semver:minor` labels (otherwise patch).
