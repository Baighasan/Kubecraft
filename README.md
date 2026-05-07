# Kubecraft

Kubecraft is a self-hosted Minecraft server platform built on Kubernetes. It lets a small group of friends each run their own isolated Minecraft server on shared infrastructure, managed entirely through a CLI tool — no web dashboard, no admin intervention after initial setup.

The platform runs on a single Oracle Cloud Ampere instance (ARM64) at $0/month using the Always Free Tier. Kubernetes handles multi-tenancy, resource isolation, and server lifecycle. Terraform provisions the infrastructure. Everything from user registration to server creation is automated.

**Stack:** Go · Kubernetes (K3s) · Terraform · Oracle Cloud · Docker

---

## Architecture

```
  User's Machine                        Oracle Cloud (OCI)
  ─────────────                         ──────────────────────────────────────────────
                                        ┌─────────────────────────────────────────┐
                                        │  VM.Standard.A1.Flex (ARM64)            │
                                        │  3 OCPU · 16GB RAM · 100GB disk         │
                                        │                                          │
  ┌───────────┐  POST /register         │  ┌──────────────────────────────────┐   │
  │           │ ──────────────────────► │  │ kubecraft-system namespace        │   │
  │ kubecraft │   :30099                │  │  Registration Service (pod)       │   │
  │   CLI     │ ◄────────────────────── │  │  - creates namespace + RBAC       │   │
  │           │   {token}               │  │  - returns 5-year SA token        │   │
  │           │                         │  └──────────────────────────────────┘   │
  │           │  K8s API calls          │                                          │
  │           │ ──────────────────────► │  ┌──────────────────────────────────┐   │
  │  uses     │   :6443 (with token)    │  │ mc-{username} namespace           │   │
  │  stored   │ ◄────────────────────── │  │  StatefulSet  ← server pod        │   │
  │  token    │                         │  │  Service      ← NodePort :3000x   │   │
  └───────────┘                         │  │  PVC          ← 10Gi world data   │   │
                                        │  └──────────────────────────────────┘   │
  Minecraft                             │                                          │
  ┌───────────┐  TCP                    │  Each user gets their own namespace.     │
  │  Client   │ ──────────────────────► │  RBAC prevents cross-namespace access.   │
  └───────────┘   :3000x (NodePort)     │                                          │
                                        └─────────────────────────────────────────┘
```

---

## How It's Built

### Infrastructure

Terraform provisions the full OCI stack: VCN, subnet, security list, and the Ampere compute instance. K3s is installed on first boot via cloud-init. Minecraft servers are exposed via **NodePort** services (ports 30000–30015) directly on the instance's public IP — no load balancer needed at this scale.

### Multi-Tenancy

Each user gets a dedicated Kubernetes namespace (`mc-{username}`) with:
- A `Role` scoped to their namespace (create/manage StatefulSets, Services, PVCs)
- A `ResourceQuota` capping them to one server and limiting CPU/memory
- A shared `ClusterRole` for read-only capacity checks across the cluster

The registration service is the only component with cluster-wide write permissions. Once a user is registered, their token only grants access to their own namespace.

### Registration Flow

1. `kubecraft register --username <name>` sends a POST to the registration service
2. Service validates the username, checks the 15-user cap, then creates the namespace, ServiceAccount, Role, RoleBinding, and ResourceQuota
3. A 5-year ServiceAccount token is generated via the TokenRequest API and returned to the CLI
4. Token is saved to `~/.kubecraft/config` — all future commands use it directly against the K8s API

### CLI

Built with Go and Cobra. The cluster endpoint and node IP are embedded at build time via `ldflags` — the binary ships pre-configured.

```
kubecraft register --username <name>   # one-time setup
kubecraft server create <name>         # pre-flight check → allocate port → wait for ready
kubecraft server list                  # name, status, NodePort, age
kubecraft server start <name>          # scale StatefulSet 0→1
kubecraft server stop <name>           # scale StatefulSet 1→0, PVC preserved
kubecraft server delete <name>         # remove StatefulSet + Service + PVC
```

Before creating a server, the CLI sums memory requests across all running pods and rejects the request if headroom drops below 4GB — preventing OOM on the shared node.

### Minecraft Servers

Each server is a StatefulSet backed by a 10Gi PVC for world persistence. The Docker image downloads the PaperMC jar at startup and is configured via environment variables (`VERSION`, `GAME_MODE`, `MAX_PLAYERS`, `JAVA_MEMORY`). Images are multi-arch (AMD64 + ARM64).

---

## Repository Layout

```
cmd/                        # Binary entrypoints (CLI + registration server)
internal/
  k8s/                      # Kubernetes API wrapper (client-go)
  registration/             # HTTP handler + username validation
  config/                   # Constants, config file management
  cli/                      # Cobra command implementations
manifests/                  # Kubernetes YAML templates
docker/                     # Dockerfiles for Minecraft server + registration service
terraform/                  # OCI infrastructure as code
.github/workflows/          # CI: unit tests, integration tests, image builds
```

---

## Deployment

```bash
# Provision OCI infrastructure
cd terraform && terraform init && terraform apply

# Build CLI pointed at the new instance
make build-prod

# Apply system manifests to the cluster
helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane
```

Integration tests run against a local k3d cluster:

```bash
make cluster-up && make cluster-setup
go test -tags=integration ./internal/...
```

---

## Status

Core implementation is complete. Waiting on Oracle Cloud capacity to provision the Ampere instance — running a polling script to claim one as it becomes available.
