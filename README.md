# Kubecraft

Kubecraft is a self-hosted Minecraft server platform for Kubernetes.
It gives each user an isolated server in their own namespace, managed entirely through a CLI tool - no web dashboard, no per-user admin work after setup.

Kubecraft is cluster-agnostic. You can run it on any Kubernetes cluster that exposes:
- Kubernetes API access for CLI clients
- A registration endpoint on NodePort `30099`
- Minecraft NodePort traffic on `30000-30015`

**Stack:** Go - Kubernetes - Helm - Terraform (optional) - Docker

---

## Architecture

**Ownership model:**
- **Terraform** owns infrastructure lifecycle (optional, provider-specific modules)
- **Helm** owns static control-plane resources (`charts/kubecraft-control-plane`)
- **Go runtime code** owns dynamic tenant and server resources

```
                           Kubernetes Cluster
  ---------------------------------------------------------------------------
  |                                                                         |
  |  kubecraft-system namespace                                             |
  |  +-------------------------------+                                      |
  |  | Registration Service          |  NodePort: 30099                    |
  |  | - validates users             |                                      |
  |  | - creates namespace + RBAC    |                                      |
  |  | - issues SA token (5y)        |                                      |
  |  +-------------------------------+                                      |
  |                                                                         |
  |  Tenant Namespaces (one per user)                                       |
  |  +---------------------------+   +---------------------------+          |
  |  | mc-alice                  |   | mc-bob                    |          |
  |  | - StatefulSet (server)    |   | - StatefulSet (server)    |          |
  |  | - Service (NodePort)      |   | - Service (NodePort)      |          |
  |  | - PVC (10Gi world data)   |   | - PVC (10Gi world data)   |          |
  |  | - Role/RoleBinding        |   | - Role/RoleBinding        |          |
  |  +---------------------------+   +---------------------------+          |
  |                                                                         |
  |  Kubernetes API Server (:6443)                                          |
  ---------------------------------------------------------------------------

  External Clients
  +---------------------------+      +----------------------------+
  | kubecraft CLI             |----->| Register + K8s API access  |
  | - init/register/server *  |      | (:30099, :6443)            |
  +---------------------------+      +----------------------------+

  +---------------------------+      +----------------------------+
  | Minecraft Client          |----->| Server NodePort access     |
  | - joins active server     |      | (:30000-30015)             |
  +---------------------------+      +----------------------------+
```

---

## Requirements

- Kubernetes cluster reachable by users running the CLI
- Cluster API endpoint available to clients (default port `6443`)
- NodePort range `30000-30015` open for Minecraft traffic
- NodePort `30099` open for registration service
- StorageClass available for PVC provisioning (`local-path` by default in K3s)
- Helm available for control-plane installation

## Quickstart

Install static control-plane resources:

```bash
helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane
```

On each user's machine:

```bash
# 1) Configure endpoint and TLS behavior
kubecraft init --ip <cluster-ip-or-dns>

# 2) Register once
kubecraft register --username <name>

# 3) Create a server
kubecraft server create <server-name>
```

## CLI Commands

```bash
kubecraft init --ip <cluster-ip-or-dns>
kubecraft register --username <name>
kubecraft server create <name>
kubecraft server list
kubecraft server start <name>
kubecraft server stop <name>
kubecraft server delete <name>
```

### Registration Flow

1. `kubecraft register --username <name>` posts to the registration service.
2. Service validates username, checks user cap, and creates namespace + RBAC + quota resources.
3. Service issues a 5-year ServiceAccount token via TokenRequest API.
4. CLI stores token at `~/.kubecraft/config` and uses it for future Kubernetes API calls.

### Multi-Tenancy Model

Each user gets namespace `mc-{username}` with:
- Namespace-scoped `Role` and `RoleBinding` for server lifecycle actions
- `ResourceQuota` limiting tenant resource usage
- Access to shared read-only capacity checks

The registration service is the only component with cluster-wide write privileges.

### Runtime Guards

- Username/server name must be lowercase alphanumeric, 3-16 chars, starting with a letter
- Capacity guard is tuned for the current single-node profile (14 Gi usable RAM): reject creates if free memory would drop below 4 Gi
- Default server resources: request 2 Gi, limit 4 Gi, PVC 10 Gi

---

## Images and Tagging

Published on GHCR:

| Image | Purpose |
|-------|---------|
| `ghcr.io/baighasan/kubecraft-minecraft` | Minecraft runtime |
| `ghcr.io/baighasan/kubecraft-registration` | Registration service |

Tag policy:
- `dev` branch push -> mutable `:dev` tag
- pull requests -> build only, no push
- release tags (`v*`) -> immutable version tags

Use pinned release tags in production.

---

## Development

### Build and test

```bash
make build
make test
```

### Integration tests (real cluster required)

```bash
go test -p 1 -tags=integration ./internal/...
```

### Local k3d flow

```bash
make cluster-up
make cluster-setup
go test -p 1 -tags=integration ./internal/...
make cluster-down
```

### Dev image overrides

```bash
# Flag
kubecraft server create myserver --server-image=ghcr.io/baighasan/kubecraft-minecraft:dev

# Environment variable
export KUBECRAFT_SERVER_IMAGE=ghcr.io/baighasan/kubecraft-minecraft:dev
kubecraft server create myserver
```

---

## Optional: OCI Reference Deployment

The `terraform/` directory contains an Oracle Cloud reference deployment for teams that want a single-node K3s host provisioned automatically.

```bash
cd terraform
terraform init
terraform apply
```

This path is optional and not required for Kubecraft itself.

---

## Repository Layout

```
cmd/                               # CLI and registration-server entrypoints
internal/
  k8s/                             # Kubernetes orchestration layer
  registration/                    # /register handler + validation
  config/                          # CLI config file + constants
  cli/                             # Cobra command implementations
charts/kubecraft-control-plane/    # Helm chart for static control-plane resources
docker/                            # Dockerfiles for Minecraft and registration images
terraform/                         # Optional OCI reference infra module
.github/workflows/                 # CI pipelines
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `Error: cannot reach Kubernetes API at https://<ip>:6443` | Firewall/network policy/security group blocks API access | Open inbound `6443` to the intended clients |
| `Warning: registration endpoint unreachable on :30099` | Helm chart not installed or NodePort unavailable | Install/upgrade `kubecraft-control-plane` and verify NodePort exposure |
| `Warning: TLS certificate verification failed. Falling back to insecure mode.` | Untrusted/self-signed API cert | Expected on many default K3s setups; re-run `kubecraft init` after trust changes |
| `Error: cluster not initialized. Run kubecraft init --ip <public-ip> first.` | Missing `~/.kubecraft/config` | Run `kubecraft init --ip <cluster-ip-or-dns>` first |

---

## Status

Static control-plane resources are Helm-managed.
Dynamic tenant and server resources are created by Go runtime code.
No operational path should apply raw Kubernetes manifests for these resources.
