# Kubecraft

Self-service Minecraft server hosting platform. Users create, manage, and connect to their own Minecraft servers through a CLI tool. Built with Kubernetes, Terraform, and Go.

**Cost:** $0/month — Oracle Cloud Always Free Tier (Ampere ARM64, 3 OCPU, 16GB RAM)

---

## Architecture

**Ownership boundary:**
- Terraform owns infrastructure lifecycle.
- Helm owns static control-plane Kubernetes resources.
- Go code owns dynamic tenant and server runtime resources.

**Single-node K3s cluster on Oracle Cloud:**
- One Ampere instance (VM.Standard.A1.Flex) provisioned by Terraform
- K3s handles orchestration; Terraform handles OCI infra
- **Networking:** NodePort services — ports 30000–30015 for Minecraft, 30099 for registration
- **Storage:** `local-path` StorageClass (HostPath on instance block volume)
- **ARM64** — all Docker images must be multi-arch (AMD64 + ARM64)

**Core components:**
1. **CLI Tool** — Go/Cobra, direct K8s API access, embedded cluster endpoint
2. **Registration Service** — Go HTTP server in `kubecraft-system` namespace (NodePort 30099)
3. **User Namespaces** — `mc-{username}`, one per user, RBAC-isolated
4. **Minecraft Pods** — StatefulSets with 10Gi PVCs, up to 1 per user
5. **System RBAC** — ClusterRole for capacity checking, shared across all users

**Data flow:**
```
Registration (one-time):
  kubecraft register  →  POST :30099/register  →  Registration Service
                                                    (creates namespace, RBAC, token)
                      ←  {token: "..."}
                         saved to ~/.kubecraft/config

Server operations (ongoing):
  kubecraft server *  →  K8s API :6443  (authenticates with stored token)
```

**Capacity:**
- 14GB usable RAM (16GB − 2GB system)
- 2Gi request / 4Gi limit per server → 3–4 concurrent servers
- Pre-flight check rejects creation if available RAM < 4GB

---

## Code Structure

Single Go module: `github.com/baighasan/kubecraft`

```
cmd/
├── kubecraft/main.go              # CLI entrypoint
└── registration-server/main.go   # HTTP server (port 8080, /register + /health)

internal/
├── k8s/
│   ├── client.go       # NewInClusterClient(), NewClientFromToken()
│   ├── namespace.go    # Create/Delete/Exists/Count namespaces
│   ├── rbac.go         # ServiceAccount, Role, RoleBinding, ResourceQuota, ClusterRoleBinding
│   ├── token.go        # 5-year ServiceAccount token via TokenRequest API
│   └── server.go       # CreateServer, DeleteServer, ListServers, ScaleServer,
│                       #   WaitForReady, CheckNodeCapacity, AllocateNodePort, GetNodePort
├── registration/
│   ├── handler.go      # /register endpoint — orchestrates registration, cleanup on failure
│   └── validator.go    # Username: 3–16 chars, alphanumeric, starts with letter
├── config/
│   ├── constants.go    # MAX_USERS=15, NodePort range, resource limits, RBAC names
│   └── config.go       # ~/.kubecraft/config — load/save Username + Token
└── cli/
    ├── root.go         # Cobra root, config loading, K8s client init
    ├── register.go     # HTTP POST to registration service, saves token
    └── server/
        ├── server.go   # Server command group
        ├── create.go   # Capacity check → allocate port → create → wait for ready
        ├── list.go     # Table output: name, status, NodePort, age
        ├── start.go    # Scale StatefulSet 0→1
        ├── stop.go     # Scale StatefulSet 1→0 (PVC preserved)
        └── delete.go   # Confirmation prompt → delete StatefulSet + Service + PVC

charts/kubecraft-control-plane/  # Helm chart for static control-plane resources
├── templates/              # namespace, serviceaccount, clusterrole, clusterrolebinding,
│                           #   deployment, service
├── values.yaml             # tunable config (image, NodePort, RBAC names)
└── Chart.yaml              # chart metadata

manifests/
├── user-templates/         # reference-only (dynamic resources created by Go code)
└── server-templates/       # reference-only (dynamic resources created by Go code)

docker/
├── minecraft/Dockerfile    # eclipse-temurin:21-jre-jammy, non-root, /data volume
├── minecraft/start.sh      # Downloads PaperMC jar, G1GC flags, env var config
└── registration/Dockerfile # Multi-stage Go build → Alpine runtime

terraform/
├── main.tf         # OCI provider, data sources (availability domain, Ubuntu 22.04 image)
├── variables.tf    # OCI credentials, SSH key, IP whitelist, region
├── network.tf      # VCN (10.0.0.0/16), subnet (10.0.1.0/24), IGW, route table
├── security.tf     # SSH from your IP, K8s API :6443, NodePorts 30000–30099
├── compute.tf      # Ampere instance, 100GB boot volume, cloud-init reference
├── outputs.tf      # Instance IP, SSH command, kubeconfig command, cluster endpoint
└── cloud-init.yaml # K3s install + anti-idle cron job (curls API every 5 min)

.github/workflows/
├── test-unit.yml          # Unit tests (no cluster needed)
├── test-integration.yml   # Integration tests with k3d cluster
├── test-manifests.yml     # YAML validation + kubectl dry-run
├── minecraft-image.yml    # Build Minecraft Docker image
└── registration-image.yml # Build registration service Docker image
```

---

## Build & Test

```bash
# CLI builds
make build-dev    # localhost endpoint, TLS insecure (for k3d)
make build-prod   # OCI instance endpoint, TLS strict

# Tests
make test                                   # Unit tests only
go test -tags=integration ./internal/...   # Integration tests (requires cluster)

# Local cluster
make cluster-up     # k3d with NodePort mapping 30000-30099
make cluster-setup  # Install control-plane Helm chart
make cluster-down   # Tear down k3d cluster
```

**Build-time variables (injected via ldflags):**
- `ClusterEndpoint` — K8s API server address embedded in CLI binary
- `NodeAddress` — public IP returned to users after server creation
- `TLSInsecure` — `true` for dev (k3d), `false` for prod

---

## Authentication Model

- No username/password login — registration generates a **5-year ServiceAccount token**
- Token stored in `~/.kubecraft/config` and used for all subsequent K8s API calls
- Registration service runs with ClusterRole (elevated); users get namespace-scoped Role only
- To use on multiple computers: copy `~/.kubecraft/config`

**ServiceAccount identities:**
- Registration service: `system:serviceaccount:kubecraft-system:registration-service`
- Users: `system:serviceaccount:mc-{username}:{username}`

---

## Key Design Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| Cloud provider | Oracle Cloud (OCI) | Always Free Tier: 3 OCPU, 16GB RAM, $0/month |
| Networking | NodePort (30000–30015) | No LoadBalancer cost; fine for small scale |
| Storage | local-path (HostPath) | Simpler, faster; tied to single node |
| Safety | CLI-side pre-flight checks | Prevents OOM from crashing the shared node |
| Auth | Long-lived SA tokens (5yr) | No cert infrastructure; easy to automate |
| CPU arch | ARM64 (Ampere) | Required for Always Free compute on OCI |

---

## Future Work

- Push multi-arch Docker images in CI (currently only builds, doesn't push)
- CLI release binaries in GitHub Actions (`cli-release.yml` not yet implemented)
- Admin user cleanup via CLI/API (runtime code path)
- Update README with quick-start guide for end users
- Optional: `kubecraft config export/import` for multi-computer convenience
