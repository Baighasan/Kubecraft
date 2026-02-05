# Minecraft Server Platform - Project Plan

## Project Overview

Self-service Minecraft server hosting platform where users can create, manage, and connect to their own Minecraft servers through a CLI tool. Built with Kubernetes, Terraform, and Go to demonstrate DevOps and platform engineering skills.

## Architecture Summary

**Single-Node Kubernetes Setup:**
- One Oracle Cloud Ampere instance (3 OCPU, 16GB RAM) running K3s — **Always Free Tier**
- All components run as pods on this single node
- Terraform provisions Oracle Cloud Infrastructure (OCI)
- K3s manages container orchestration
- **Networking:** `NodePort` services exposing ports 30000-30015 directly on the host IP
- **Storage:** `local-path` StorageClass writing directly to the instance's block volume (HostPath)
- **Architecture:** ARM64 (Ampere) — requires multi-arch Docker images

**Core Components:**
1. **CLI Tool**: Go-based command-line interface with direct Kubernetes API access and **pre-flight capacity checks**
2. **Registration Service**: HTTP service for self-service user account creation (NodePort 30099)
3. **User Namespaces**: One namespace per user with RBAC-enforced isolation
4. **Minecraft Pods**: StatefulSets within user namespaces (up to 1 per user)
5. **System Namespace**: Registration service and optional monitoring tools

**Tech Stack:**
- Infrastructure: Oracle Cloud (Compute, VCN), Terraform, K3s
- Application Code: Go 1.25.5 with monorepo structure (single module, multiple binaries)
- CLI Tool: Go with Cobra framework, client-go (Kubernetes client library)
- Registration Service: Go HTTP server, client-go with elevated permissions
- Container Orchestration: K3s (lightweight Kubernetes)
- Authentication: Kubernetes RBAC + ServiceAccount tokens (5-year expiration)
- CI/CD: GitHub Actions
- Container Registry: Docker Hub (multi-arch images for AMD64 + ARM64)

> **📚 For detailed code structure and package responsibilities, see [Code Structure & Architecture](#code-structure--architecture) section below.**

## Project Scope

**Users:** 5 people
**Servers per user:** Up to 1
**Concurrent servers:** 3-4 running simultaneously (with 14GB RAM available for workloads)
**Total servers:** Up to 15 (most stopped to save resources)
**Monthly cost:** $0 (Oracle Cloud Always Free Tier — Ampere with 100GB storage)

---

## Code Structure & Architecture

### Go Project Layout (Monorepo)

The project uses a **single Go module** with multiple binaries:

```
kubecraft/                                    # Repository root
│
├── go.mod                                    # Single module: github.com/baighasan/kubecraft
├── go.sum                                    # Dependency checksums
│
├── cmd/                                      # Binary entrypoints
│   ├── kubecraft/                            # CLI tool (Phase 3)
│   │   └── main.go                           # User-facing CLI
│   │
│   └── registration-server/                  # Registration service (Phase 2.5)
│       └── main.go                           # HTTP server for self-service registration
│
├── internal/                                 # Shared libraries (internal to this module)
│   ├── k8s/                                  # Kubernetes API wrapper (shared)
│   │   ├── client.go                         # Initialize K8s clientset
│   │   ├── namespace.go                      # Namespace operations
│   │   ├── rbac.go                           # Create ServiceAccounts, Roles, RoleBindings
│   │   ├── token.go                          # Generate ServiceAccount tokens
│   │   └── server.go                         # StatefulSet/Service operations (for CLI)
│   │
│   ├── registration/                         # Registration business logic
│   │   ├── handler.go                        # HTTP /register endpoint
│   │   └── validator.go                      # Username validation
│   │
│   ├── config/                               # Shared configuration
│   │   └── constants.go                      # Cluster endpoint, MAX_USERS, ports
│   │
│   └── cli/                                  # CLI command implementations
│       ├── root.go                           # Root command and global flags
│       ├── register.go                       # Register command
│       └── server/                           # Server subcommands
│           ├── server.go                     # Server command group
│           ├── create.go                     # Server create command
│           ├── list.go                       # Server list command
│           ├── start.go                      # Server start command
│           ├── stop.go                       # Server stop command
│           └── delete.go                     # Server delete command
│
├── manifests/                                # Kubernetes YAML templates
│   ├── user-templates/                       # Per-user resources
│   ├── server-templates/                     # Minecraft server manifests
│   ├── registration-templates/               # Registration service deployment
│   └── system-templates/                     # Cluster-wide RBAC
│
├── docker/                                   # Container images
│   ├── Dockerfile                            # Minecraft server image
│   └── start.sh                              # Minecraft startup script
│
├── terraform/                                # Oracle Cloud infrastructure as code
│   ├── main.tf                               # Provider config, data sources
│   ├── network.tf                            # VCN, subnet, internet gateway
│   ├── compute.tf                            # Ampere A1 instance, cloud-init
│   ├── security.tf                           # Security list (firewall rules)
│   ├── variables.tf                          # Input variables
│   ├── outputs.tf                            # Instance IP, connection info
│   └── cloud-init.yaml                       # K3s installation script
│
├── Makefile                                  # Build automation (dev/prod targets, cluster management)
│
├── scripts/                                  # Admin helper scripts
│   └── delete-user.sh                        # Manual user cleanup
│
└── .github/workflows/                        # CI/CD pipelines
    ├── minecraft-image.yml
    ├── registration-image.yml
    ├── test-unit.yml
    ├── test-integration.yml
    └── cli-release.yml
```

### Package Responsibilities

**`cmd/kubecraft/`** - CLI Tool (Phase 3)
- **Purpose:** User-facing command-line interface entrypoint
- **Used by:** End users on their local computers
- **Talks to:** Kubernetes API directly (using tokens from registration)
- **Commands:** `register`, `server create`, `server list`, `server start`, `server stop`, `server delete`
- **Dependencies:** Imports `internal/cli`, `internal/k8s`, `internal/config`

**`cmd/registration-server/`** - Registration Service (Phase 2.5)
- **Purpose:** HTTP server for self-service user onboarding
- **Used by:** CLI's `register` command (one-time interaction)
- **Runs on:** OCI instance as a pod in `kubecraft-system` namespace
- **Exposes:** NodePort 30099 for HTTP endpoint
- **Permissions:** Elevated (ClusterRole) to create namespaces and RBAC
- **Dependencies:** Imports `internal/k8s`, `internal/registration`, `internal/config`

**`internal/k8s/`** - Kubernetes Operations (Shared Library)
- **Purpose:** Wrapper around client-go for common K8s operations
- **Used by:** Both CLI and registration service
- **Provides:**
  - `client.go` - Initialize Kubernetes clientset (InClusterConfig, token-based, or rest.Config). Supports TLS skip for dev via build-time `TLSInsecure` variable
  - `namespace.go` - Create/check namespaces, count users
  - `rbac.go` - Create ServiceAccounts, Roles, RoleBindings, patch ClusterRoleBindings
  - `token.go` - Generate ServiceAccount tokens via TokenRequest API
  - `server.go` - ServerExists, CreateServer, DeleteServer, ListServers, ScaleServer, WaitForReady, CheckNodeCapacity, AllocateNodePort, GetNodePort

**`internal/registration/`** - Registration Business Logic
- **Purpose:** HTTP endpoint handling and validation
- **Used by:** Registration service only
- **Provides:**
  - `handler.go` - Parse HTTP requests, orchestrate registration flow, return JSON
  - `validator.go` - Username format validation, reserved name checks

**`internal/config/`** - Shared Configuration
- **Purpose:** Constants and configuration file management
- **Used by:** Both CLI and registration service
- **Provides:**
  - `constants.go` - MAX_USERS (15), NodePort range (30000-30015), server resource limits, readiness polling config
  - `config.go` - Config struct (Username, Token), load/save/validate from `~/.kubecraft/config`
  - Build-time variables: `ClusterEndpoint` (injected via ldflags), `TLSInsecure` (dev only)

**`internal/cli/`** - CLI Command Implementations
- **Purpose:** Cobra command implementations for the CLI tool
- **Used by:** CLI entrypoint only
- **Provides:**
  - `root.go` - Root command setup, global flags, config loading
  - `register.go` - User registration via HTTP to registration service
  - `server/*.go` - Server management commands (create, list, start, stop, delete)

### Import Path Structure

All packages import using the module path:

```go
// In cmd/registration-server/main.go
package main

import (
    "github.com/baighasan/kubecraft/internal/k8s"
    "github.com/baighasan/kubecraft/internal/registration"
    "github.com/baighasan/kubecraft/internal/config"
)

// In cmd/kubecraft/main.go
package main

import (
    "github.com/baighasan/kubecraft/internal/cli"
)

// In internal/cli/root.go
package cli

import (
    "github.com/baighasan/kubecraft/internal/k8s"
    "github.com/baighasan/kubecraft/internal/config"
)

// In internal/registration/handler.go
package registration

import (
    "github.com/baighasan/kubecraft/internal/k8s"
    "github.com/baighasan/kubecraft/internal/config"
)
```

### Build Commands

```bash
# Build CLI tool (dev — TLS insecure, local cluster endpoint)
make build-dev

# Build CLI tool (prod — TLS strict, OCI instance endpoint)
make build-prod

# Build with custom endpoint
make build-dev DEV_ENDPOINT=192.168.1.100:6443

# Build registration service
go build -o bin/registration-server ./cmd/registration-server

# Local k3d cluster management
make cluster-up
make cluster-down

# Run tests
make test

# Add/update dependencies
go mod tidy
```

### Data Flow Architecture

**Registration Flow (One-Time):**
```
User's Computer                    OCI Instance (Kubernetes Cluster)
┌─────────────┐                   ┌──────────────────────────────────┐
│   kubecraft │ ──HTTP POST──────>│  Registration Service (Pod)      │
│   CLI       │   (port 30099)    │  - Validates username            │
│             │                   │  - Creates namespace             │
│             │                   │  - Creates RBAC resources        │
│             │                   │  - Generates SA token            │
│             │ <──JSON Response──│  - Returns token                 │
│             │   {token: "..."}  └──────────────────────────────────┘
│             │                              │
│   Saves to  │                              ▼
│  ~/.kubecraft/config                  Kubernetes API
└─────────────┘                         (creates resources)
```

**Server Operations Flow (Ongoing):**
```
User's Computer                    OCI Instance (Kubernetes Cluster)
┌─────────────┐                   ┌──────────────────────────────────┐
│   kubecraft │                   │                                  │
│   CLI       │ ──────────────────│──> Registration Service         │
│             │  (NOT involved)   │     (bypassed)                   │
│             │                   └──────────────────────────────────┘
│  Reads token│
│  from config│                   ┌──────────────────────────────────┐
│             │ ──K8s API Call───>│  Kubernetes API Server           │
│             │  (uses token)     │  - Authenticates token           │
│             │                   │  - Checks RBAC permissions       │
│             │ <──Response───────│  - Creates/manages resources     │
└─────────────┘                   └──────────────────────────────────┘
```

### Authentication Model

**Registration Service:** Uses ServiceAccount with ClusterRole permissions
```yaml
# Runs as: system:serviceaccount:kubecraft-system:registration-service
# Permissions: Create namespaces, RBAC resources, generate tokens
```

**CLI Tool:** Uses user's ServiceAccount token (stored in ~/.kubecraft/config)
```yaml
# Authenticates as: system:serviceaccount:mc-{username}:{username}
# Permissions: Limited to user's namespace (create servers, manage their resources)
```

**Key Insight:** Registration service is a **trusted system component** with elevated permissions. Users never get these permissions - they only get namespace-scoped access.

---

## Phase 0: Prerequisites & Setup (Local First)

### Goals
- Set up development environment
- **Shift Left:** Develop and test entirely on local clusters before deploying to Oracle Cloud
- Learn foundational concepts

### What to Learn

**Git basics:**
- commit, push, pull, branches
- .gitignore patterns
- Meaningful commit messages

**Docker fundamentals:**
- Containers vs VMs
- Dockerfile syntax
- docker build, run, compose
- Image layers and caching
- Multi-stage builds
- **Multi-architecture builds** (AMD64 + ARM64)

**Kubernetes concepts (in-depth):**
- Pods, StatefulSets, Services, Namespaces
- **NodePort Networking** vs LoadBalancers
- RBAC (Roles, RoleBindings)
- PersistentVolumeClaims (local-path)
- ResourceQuotas & Limits
- Labels and selectors

**Oracle Cloud Infrastructure (OCI) basics:**
- Compute instances and shapes (VM.Standard.A1.Flex = Ampere ARM)
- Virtual Cloud Networks (VCN), subnets, and networking
- **Security Lists** (equivalent to AWS Security Groups)
- Reserved Public IPs
- **Always Free Tier** resources and limits
- Compartments and IAM

**Go basics:**
- Syntax and idioms
- Package management (go mod)
- Building binaries
- Working with structs and interfaces
- Error handling patterns
- Goroutines and channels (basic)

### Deliverables
- [ ] Development environment configured (Docker, kubectl, Terraform, Go installed)
- [ ] Oracle Cloud, GitHub, Docker Hub accounts created
- [ ] OCI CLI configured with API keys
- [ ] Project repository initialized with proper directory structure
- [ ] **Local K3s cluster running (k3d or minikube)** for Phase 1-3 development
- [ ] Basic understanding of Docker and K8s concepts validated

---

## Phase 1: Kubernetes Manifests & RBAC Setup

### Goals
- Create K8s YAML manifests for infrastructure
- Implement RBAC for multi-user isolation
- Test user namespace isolation locally
- **Switch networking model to NodePort**

### Manifests Overview

All manifests are located in the `manifests/` directory, organized by purpose. These are templates with placeholders (e.g., `{username}`, `{servername}`) that get populated dynamically by the registration service or CLI tool.

**📁 manifests/user-templates/** - Per-user namespace resources (created during registration)
- **namespace.yaml** - Creates `mc-{username}` namespace with `app: kubecraft` label
- **serviceaccount.yaml** - ServiceAccount for user authentication
- **resourcequota.yaml** - Enforces compute limits per user:
  - CPU: 1000m request, 1500m limit
  - Memory: 2Gi request, 4Gi limit
  - PVCs: max 1 per namespace
- **role.yaml** - `minecraft-manager` Role granting permissions for:
  - PVCs and Services (create, delete, get, list)
  - StatefulSets (create, get, list, patch, update, delete)
  - Pods and pod logs (read-only)
- **rolebinding.yaml** - Binds the Role to the user's ServiceAccount

**📁 manifests/server-templates/** - Minecraft server resources (created by CLI)
- **statefulset.yaml** - Minecraft server StatefulSet with:
  - Container image: `hasanbaig786/kubecraft`
  - Resource requests: 2Gi RAM, 1000m CPU
  - Resource limits: 4Gi RAM, 1500m CPU
  - Volume mount: `/data` (for world persistence, 10Gi storage)
  - Readiness probe: TCP socket on port 25565
  - Environment variables: VERSION, GAME_MODE, MAX_PLAYERS, EULA
- **service.yaml** - NodePort Service exposing port 25565 (NodePort auto-assigned or manually set 30000-30015)

**📁 manifests/system-templates/** - Cluster-wide RBAC (applied once by admin)
- **clusterrole.yaml** - `kc-capacity-checker` ClusterRole for pre-flight checks:
  - Read-only access to namespaces, services, pods (for capacity validation)
- **clusterrolebinding.yaml** - `kc-users-capacity-check` binding:
  - Subjects populated dynamically during user registration
  - Grants all users capacity-checking permissions

**📁 manifests/registration-templates/** - Registration service infrastructure (applied once by admin)
- **registration-namespace.yaml** - `kubecraft-system` namespace for system components
- **registration-serviceaccount.yaml** - ServiceAccount for the registration service pod
- **registration-clusterrole.yaml** - `kc-registration-admin` ClusterRole with elevated permissions:
  - Create namespaces, ServiceAccounts, Roles, RoleBindings, ResourceQuotas
  - Update ClusterRoleBindings (to add new users)
  - Generate ServiceAccount tokens
- **registration-clusterrolebinding.yaml** - Binds ClusterRole to registration ServiceAccount
- **registration-deployment.yaml** - Deployment for registration HTTP service:
  - Replicas: 1
  - Resource requests: 128Mi RAM, 100m CPU
  - Resource limits: 256Mi RAM, 200m CPU
  - Environment: MAX_USERS=15
- **registration-service.yaml** - NodePort 30099 Service for CLI registration endpoint

**CAPACITY PLANNING NOTE:**
```
Oracle Ampere total: 16GB RAM, 3 OCPU (Always Free Tier - US East Ashburn)
System overhead (K3s, OS, monitoring): ~2GB RAM, ~0.5 OCPU
Available for workloads: ~14GB RAM, ~2.5 OCPU

Per-server resources (UPGRADED for better performance):
  requests: 2Gi RAM, 1000m CPU
  limits: 4Gi RAM, 1500m CPU
  Java heap: 3G (4x more than before)
  Storage: 10Gi (2x more than before)

Capacity calculation:
  At requests (7 servers): 7 × 2Gi = 14GB RAM
  At limits (3 servers): 3 × 4Gi = 12GB RAM
  Realistic concurrent: 3-4 servers running simultaneously

Safety: Pre-flight check prevents server creation if available RAM < 4GB
```

### What to Learn

**Kubernetes RBAC:**
- Role vs ClusterRole (namespace-scoped vs cluster-scoped)
- RoleBinding vs ClusterRoleBinding
- Service accounts vs user accounts
- Principle of least privilege
- Testing RBAC policies with `kubectl auth can-i`
- RBAC API groups and resources
- Verb permissions (get, list, create, update, delete, patch, watch)

**Kubernetes Authentication:**
- ServiceAccount tokens (JWT)
- Token lifecycle and expiry
- Kubeconfig file structure
- Token-based authentication vs certificates

**Kubernetes YAML Syntax:**
- apiVersion, kind, metadata, spec structure
- Labels and selectors
- Resource organization and naming conventions
- Template variables and substitution

**Core K8s Concepts:**
- **StatefulSet**: Stable network identity, ordered pod management, persistent storage
- **Service**:
  - NodePort: Mapping internal ports to host ports
  - Why not LoadBalancer: Cost implications and architectural fit
- **PersistentVolumeClaim**:
  - local-path storage class (HostPath)
  - Dynamic provisioning on single nodes
- **ResourceQuota**: Enforcing limits per namespace to prevent OOM
- **Namespace**: Multi-tenancy isolation

**kubectl Commands:**
- `kubectl apply -f <file>`
- `kubectl get <resource> -n <namespace>`
- `kubectl describe <resource> <name> -n <namespace>`
- `kubectl logs <pod> -n <namespace>`
- `kubectl exec -it <pod> -n <namespace> -- /bin/bash`
- `kubectl delete <resource> <name> -n <namespace>`
- `kubectl auth can-i <verb> <resource> --as=<user> -n <namespace>`
- `kubectl config view`

### Deliverables
- [ ] Complete K8s manifests for user namespaces with RBAC (as templates)
- [ ] Minecraft server template manifests using NodePort
- [ ] Registration service manifests (RBAC, Deployment, Service)
- [ ] Successfully deployed and tested in local K3s cluster (k3d)
- [ ] Can create isolated namespaces with working RBAC
- [ ] RBAC policies tested with `kubectl auth can-i`

---

## Phase 2: Minecraft Server Docker Image

### Goals
- Create a custom Minecraft server Docker image
- **Build multi-architecture images** (AMD64 for local dev, ARM64 for Oracle Cloud)
- Configure for Kubernetes deployment
- Test locally

### Dockerfile Structure

Standard structure as previously defined, ensuring Volume mount points align with PVC.

### What to Learn

**Dockerfile Best Practices:**
- Multi-stage builds
- Layer caching
- **Multi-architecture builds with docker buildx**

**Multi-Arch Docker Builds:**
```bash
# Set up buildx for multi-arch
docker buildx create --name multiarch --use

# Build and push multi-arch image
docker buildx build --platform linux/amd64,linux/arm64 \
  -t hasanbaig786/kubecraft:latest --push .
```

**Minecraft Server Configuration:**
- server.properties
- EULA
- Paper vs Vanilla

**Java Memory Management:**
- G1GC
- -Xms/-Xmx
- Container limits vs JVM heap

**Docker Build & Push:**
- Tagging and Registry management
- Multi-arch manifests

### Deliverables
- [ ] Working Dockerfile for Minecraft server (multi-arch compatible)
- [ ] Startup script with environment variable configuration
- [ ] Image built and tested locally (AMD64)
- [ ] **Multi-arch image pushed to Docker Hub** (AMD64 + ARM64)
- [ ] Health check implemented and tested

---

## Phase 2.5: Registration Service

### Goals
- Build HTTP service for self-service user registration
- Automate namespace and RBAC creation
- Generate ServiceAccount tokens programmatically
- Test registration flow locally

### Architecture

**Registration Service:**
- Go HTTP server running as a pod in `kubecraft-system` namespace
- Exposes `/register` endpoint on NodePort 30099
- Has elevated permissions (ClusterRole) to create namespaces and RBAC resources
- Validates usernames, enforces user limits, generates tokens

**Authentication Flow:**
1. User runs `kubecraft register --username alice`
2. CLI sends HTTP POST to registration service
3. Service creates namespace, ServiceAccount, RBAC resources
4. Service generates long-lived token (5 years)
5. Service returns token to CLI
6. CLI saves token to `~/.kubecraft/config`

### What to Learn

**Go HTTP Servers:**
- `net/http` package
- JSON encoding/decoding
- HTTP request handling
- Graceful shutdown patterns

**Advanced client-go:**
- In-cluster configuration (`rest.InClusterConfig()`)
- Creating resources programmatically (Namespaces, Roles, RoleBindings, etc.)
- Updating existing resources (patching ClusterRoleBinding)
- TokenRequest API for ServiceAccount token generation

**ServiceAccount Tokens:**
- JWT structure and claims
- Token expiration and lifecycle
- Token-based authentication vs certificates
- Security considerations (storage, sharing, revocation)

**API Design:**
- RESTful endpoints
- Error handling and status codes
- Input validation
- Idempotency (handling duplicate registrations)

### Implementation Roadmap

**Files to Implement (in order):**

1. **`internal/config/constants.go`** - Define constants
   - MAX_USERS = 15
   - REGISTRATION_PORT = 8080
   - NODEPORT_MIN = 30000, NODEPORT_MAX = 30015
   - Reserved usernames list

2. **`internal/k8s/client.go`** - Kubernetes client initialization
   - `NewClient()` - Create clientset with InClusterConfig
   - Error handling for API server connection

3. **`internal/k8s/namespace.go`** - Namespace operations
   - `CreateNamespace(username)` - Create mc-{username} with labels
   - `NamespaceExists(username)` - Check if already exists
   - `CountUserNamespaces()` - Count namespaces for user limit check

4. **`internal/k8s/rbac.go`** - RBAC resource creation
   - `CreateServiceAccount(namespace, username)` - Create SA
   - `CreateRole(namespace)` - Create minecraft-manager Role
   - `CreateRoleBinding(namespace, username)` - Bind SA to Role
   - `UpdateClusterRoleBinding(username, namespace)` - Add user to capacity checker
   - `CreateResourceQuota(namespace)` - Apply compute limits

5. **`internal/k8s/token.go`** - Token generation
   - `GenerateToken(namespace, serviceAccountName)` - Use TokenRequest API
   - Set 5-year expiration (157680000 seconds)

6. **`internal/registration/validator.go`** - Username validation
   - `ValidateUsername(username)` - Check format (alphanumeric, 3-16 chars)
   - Check against reserved names (system, admin, root, etc.)
   - Return descriptive errors

7. **`internal/registration/handler.go`** - HTTP endpoint
   - `RegisterHandler(w http.ResponseWriter, r *http.Request)` - Main handler
   - Parse JSON request body
   - Orchestrate registration flow (validate → check limit → create resources → generate token)
   - Return JSON response with token

8. **`cmd/registration-server/main.go`** - HTTP server
   - Set up HTTP routes (`/register`, `/health`)
   - Start server on port 8080
   - Graceful shutdown handling

9. **Docker image** - Build and test
   - Multi-stage Dockerfile for registration service
   - Test locally, push to Docker Hub

### Deliverables
- [x] All `internal/k8s/` files implemented (client, namespace, rbac, token, server)
- [x] All `internal/registration/` files implemented (validator, handler)
- [x] All `internal/config/` files implemented (constants, config)
- [x] `cmd/registration-server/main.go` HTTP server running
- [x] Unit tests for registration CLI command
- [x] Integration tests for registration CLI command and k8s operations
- [ ] Dockerfile for registration service created
- [ ] Service can create all user resources (namespace, RBAC, ResourceQuota)
- [ ] Service generates valid ServiceAccount tokens (5-year expiration)
- [ ] Username validation working (format, uniqueness, reserved names)
- [ ] User limit enforcement (max 15 users)
- [ ] Tested locally with curl and in k3d cluster
- [ ] RBAC isolation verified (users can't access each other's namespaces)
- [ ] Image pushed to Docker Hub

---

## Phase 3: CLI Tool with Kubernetes Client

### Goals
- Build Go-based CLI tool with direct Kubernetes API access
- Implement self-service registration command
- Implement Pre-flight Capacity Checks (Safety Mechanism)
- Implement Polling Logic for NodePort retrieval
- Embed cluster endpoint in CLI binary

### Implementation Roadmap

**Files to Implement:**

1. **`internal/cli/root.go`** - Root command and configuration
   - Initialize Cobra root command
   - Load configuration from `~/.kubecraft/config`
   - Set up global flags (verbose, config path)
   - Initialize K8s client from stored token

2. **`internal/cli/register.go`** - Registration command
   - Send HTTP POST to registration service with username
   - Parse JSON response containing token
   - Save token to `~/.kubecraft/config`
   - Display success message with next steps

3. **`internal/cli/server/server.go`** - Server command group
   - Parent command for server subcommands
   - Shared validation (ensure user is registered)

4. **`internal/cli/server/create.go`** - Create server command
   - Validate server name format
   - Run pre-flight capacity check before creation
   - Create PVC, StatefulSet, and Service via K8s API
   - Allocate NodePort from reserved range (30000-30015)
   - Poll until pod is ready, then display connection info

5. **`internal/cli/server/list.go`** - List servers command
   - Query StatefulSets in user's namespace
   - Display table with server name, status, NodePort, age

6. **`internal/cli/server/start.go`** - Start server command
   - Scale StatefulSet replicas from 0 to 1
   - Poll until pod is ready

7. **`internal/cli/server/stop.go`** - Stop server command
   - Scale StatefulSet replicas from 1 to 0
   - Preserves PVC data for later restart

8. **`internal/cli/server/delete.go`** - Delete server command
   - Prompt for confirmation
   - Delete StatefulSet, Service, and PVC
   - Warn that world data will be permanently lost

9. **`internal/k8s/server.go`** - Server management operations
   - `CheckNodeCapacity()` - Pre-flight RAM check before server creation
   - `AllocateNodePort()` - Find first available port in 30000-30015 range
   - `CreateServer()` - Create Service and StatefulSet (with cleanup on failure)
   - `DeleteServer()` - Remove StatefulSet, Service, and PVC
   - `ListServers()` - Get all servers in namespace with status, port, age
   - `ScaleServer()` - Start/stop by scaling replicas (0 or 1)
   - `WaitForReady()` - Poll until pod is running and ready
   - `ServerExists()` - Check if StatefulSet exists
   - `GetNodePort()` - Look up a single server's NodePort

**Reuses from Phase 2.5:**
- `internal/k8s/client.go` - Already implemented for registration service
- `internal/k8s/namespace.go` - For capacity checks (counting namespaces)
- `internal/config/constants.go` - Shared constants

### Authentication Architecture

**No Traditional Login:**
- Users don't "sign in" with username/password
- Registration generates a long-lived token (5 years)
- Token stored in `~/.kubecraft/config` (like a permanent credential)
- Every CLI command reads token from disk and sends to Kubernetes

**Multi-Computer Usage:**
- Users copy `~/.kubecraft/config` to other machines
- Optional: Add `kubecraft config export/import` commands for convenience

**Token Expiry:**
- After 5 years, token becomes invalid
- User's namespace and data persist in cluster
- User must contact admin to reset account (or add token refresh feature later)

### Key Features to Implement

**Pre-flight Capacity Check:**
- Before creating a server, sum memory requests from all running Minecraft pods
- Compare against available RAM (16GB total - 2GB system overhead = 14GB available)
- Reject creation if insufficient capacity, with helpful error message
- Prevents OOM situations that would crash the node

**NodePort Allocation:**
- Scan all Services across all `mc-*` namespaces
- Find first unused port in 30000-30015 range
- Explicitly assign port when creating Service
- Prevents port collisions between users

**Server Readiness Polling:**
- After creating StatefulSet, poll for pod status
- Check both `phase=Running` and `Ready` condition
- Timeout after 5 minutes with helpful error
- Return NodePort for user to connect

### What to Learn

**Go Programming:**
- Polling patterns
- Safety checks
- "Fail Fast" logic
- HTTP clients (making POST requests)
- File I/O (reading/writing kubeconfig)

**CLI Libraries:**
- Cobra for command structure
- Flag parsing and validation
- User-friendly output formatting

**client-go Library:**
- Typed clients
- Label selectors
- Watching resources
- Kubeconfig loading and management

**Build-time Configuration:**
- Embedding constants (cluster endpoint)
- Version information in binaries

### Deliverables
- [x] `kubecraft register` command implemented
- [x] Cluster endpoint injected at build time via ldflags (no --cluster flag needed)
- [x] CLI sends HTTP request to registration service
- [x] CLI saves received token to `~/.kubecraft/config`
- [x] CLI reads token from config for all server commands
- [x] CLI tool implements "Pre-flight" memory check
- [x] `create` command waits for Pod to be Ready
- [x] `create` command returns the specific NodePort (e.g., 30001) to the user
- [x] All server subcommands implemented (create, list, start, stop, delete)
- [x] Unit tests for server name validation and age formatting
- [x] Integration tests for k8s server operations
- [x] Makefile for dev/prod builds and local cluster management
- [ ] Optional: `config export/import` commands for multi-computer usage

---

## Phase 4: Terraform Infrastructure (Oracle Cloud)

### Goals
- Define Oracle Cloud infrastructure as code using Terraform
- Provision Ampere instance with K3s (Always Free Tier)
- Configure networking and firewall rules for NodePort access
- Automate K3s installation via cloud-init

### Infrastructure Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                  Oracle Cloud (us-ashburn-1)                     │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │              VCN: kubecraft-vcn (10.0.0.0/16)              │ │
│  │                                                            │ │
│  │  ┌──────────────────────────────────────────────────────┐ │ │
│  │  │         Subnet: kubecraft-public (10.0.1.0/24)       │ │ │
│  │  │                                                      │ │ │
│  │  │  ┌────────────────────────────────────────────────┐ │ │ │
│  │  │  │     Compute Instance: kubecraft-k3s            │ │ │ │
│  │  │  │     • Shape: VM.Standard.A2.Flex (ARM64)       │ │ │ │
│  │  │  │     • CPU: 3 OCPU                              │ │ │ │
│  │  │  │     • RAM: 16 GB                               │ │ │ │
│  │  │  │     • Disk: 100 GB                             │ │ │ │
│  │  │  │     • OS: Ubuntu 22.04                         │ │ │ │
│  │  │  │     • K3s installed via cloud-init             │ │ │ │
│  │  │  └────────────────────────────────────────────────┘ │ │ │
│  │  └──────────────────────────────────────────────────────┘ │ │
│  │                          │                                 │ │
│  │              Route Table → Internet Gateway                │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Terraform File Structure

```
terraform/
├── main.tf              # Provider config, data sources (availability domain, Ubuntu image)
├── variables.tf         # Input variables (OCI credentials, SSH key, IP whitelist)
├── network.tf           # VCN, subnet, internet gateway, route table
├── security.tf          # Security list (firewall rules for SSH, K8s API, NodePorts)
├── compute.tf           # Ampere instance configuration
├── outputs.tf           # Instance IP, SSH command, cluster endpoint
├── cloud-init.yaml      # K3s installation script (runs on first boot)
├── terraform.tfvars     # Your actual values (DO NOT commit to git)
└── .gitignore           # Ignore tfvars and state files
```

### Resources Created by Terraform

| Resource | Purpose |
|----------|---------|
| **VCN** | Virtual network (10.0.0.0/16) isolating our infrastructure |
| **Subnet** | Public subnet (10.0.1.0/24) where the instance lives |
| **Internet Gateway** | Allows outbound internet access |
| **Route Table** | Routes 0.0.0.0/0 traffic to internet gateway |
| **Security List** | Firewall rules (SSH, K8s API, NodePorts) |
| **Compute Instance** | Ampere ARM64 VM running K3s |

### Security List Rules (Firewall)

| Port | Protocol | Source | Purpose |
|------|----------|--------|---------|
| 22 | TCP | Your IP only | SSH access |
| 6443 | TCP | 0.0.0.0/0 | Kubernetes API (kubectl) |
| 30000-30099 | TCP | 0.0.0.0/0 | Minecraft servers + Registration service |

### Oracle Cloud Always Free Tier Limits (US East Ashburn)

| Resource | Free Tier Limit | Our Usage |
|----------|-----------------|-----------|
| Ampere Compute | 3 OCPU, 16GB RAM | 3 OCPU, 16GB (1 instance) |
| Block Storage | 200GB total | 100GB boot volume |
| Outbound Data | 10TB/month | Minimal (~5GB/month) |
| VCN | 2 VCNs | 1 VCN |

### Free Tier Caveats

1. **Idle Reclamation:** Oracle may reclaim idle instances after 7 days — solved with a cron job
2. **Region Limits:** Free tier specs vary by region; Ashburn offers A2/A3 shapes with max 3 OCPU/16GB
3. **Account Verification:** Credit card required but won't be charged for Always Free resources

### What to Learn

| Topic | Key Concepts |
|-------|--------------|
| **Terraform Basics** | Providers, resources, variables, outputs, state |
| **HCL Syntax** | Blocks, attributes, references, interpolation |
| **OCI Networking** | VCN, subnets, security lists, internet gateway |
| **Cloud-init** | First-boot automation, package installation, scripts |

### Deployment Steps

1. **Prerequisites:** Install Terraform, OCI CLI; create API keys in OCI console
2. **Configure:** Copy `terraform.tfvars.example` to `terraform.tfvars`, fill in values
3. **Initialize:** `terraform init` (downloads OCI provider)
4. **Preview:** `terraform plan` (shows what will be created)
5. **Apply:** `terraform apply` (creates infrastructure)
6. **Verify:** SSH into instance, confirm K3s is running
7. **Get kubeconfig:** Copy `/etc/rancher/k3s/k3s.yaml` to local machine

### Deliverables
- [ ] OCI account created with API keys configured
- [ ] Terraform files written and tested
- [ ] VCN, Subnet, Security List provisioned
- [ ] Ampere instance (3 OCPU, 16GB RAM) running
- [ ] K3s installed and accessible via kubectl
- [ ] Security List opens ports 22, 6443, 30000-30099
- [ ] Anti-idle cron job configured

---

## Phase 5: System Deployment

### Goals
- Deploy registration service to cluster
- Create system namespace and RBAC
- Verify self-service registration works end-to-end

### One-Time Admin Setup

**Initialize System Components:**
1. Create `kubecraft-system` namespace
2. Apply cluster-wide RBAC (ClusterRole, ClusterRoleBinding for capacity checks)
3. Deploy registration service (RBAC, Deployment, Service)
4. Verify registration service is accessible at `http://OCI_IP:30099`

**Admin Maintenance:**
- `delete-user.sh` script for removing users if needed
- Monitoring registration service logs
- No per-user manual work required

### What to Learn

**Kubernetes System Services:**
- System namespaces vs user namespaces
- Service discovery within cluster
- In-cluster configuration for pods
- Health checks and liveness probes

**Deployment Best Practices:**
- Resource limits for system services
- Graceful shutdown handling
- Log aggregation and monitoring

### Deliverables
- [ ] `kubecraft-system` namespace created
- [ ] Registration service deployed and running (ARM64 image)
- [ ] Registration service accessible via NodePort 30099
- [ ] Can successfully register users via CLI
- [ ] Tokens generated are valid and work for server management
- [ ] User limit enforcement working (rejects 16th user)
- [ ] Admin deletion script for cleanup
- [ ] Documentation for admin deployment process

---

## Phase 6: CI/CD Pipeline

### Goals
- Automate building and pushing Docker images **(multi-arch: AMD64 + ARM64)**
- Automate CLI binary builds and releases
- Ensure cluster endpoint is updated in CLI on deployment

### GitHub Actions Workflows

**Existing:**
- `minecraft-image.yml` - Builds and pushes Minecraft server image **(multi-arch)**
- `cli-release.yml` - Builds CLI binaries for multiple platforms
- `test-unit.yml` - Runs unit tests (config, registration, cli, cli/server)
- `test-integration.yml` - Runs integration tests with k3d cluster (k8s, cli)
- `registration-image.yml` - Builds and pushes registration service image **(multi-arch)**

**Multi-Arch Docker Build Example:**
```yaml
# In GitHub Actions workflow
- name: Set up QEMU
  uses: docker/setup-qemu-action@v3

- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Build and push
  uses: docker/build-push-action@v5
  with:
    context: .
    platforms: linux/amd64,linux/arm64
    push: true
    tags: hasanbaig786/kubecraft:latest
```

**Build-time Configuration:**
- CLI build embeds current cluster endpoint (update when OCI IP changes)
- Version tagging for releases

### Deliverables
- [ ] Working CI/CD pipeline for Minecraft image **(multi-arch)**
- [ ] Working CI/CD pipeline for registration service image **(multi-arch)**
- [ ] Automated CLI builds for multiple platforms (Linux AMD64, Linux ARM64, macOS, Windows)
- [ ] GitHub releases created on version tags
- [ ] Documentation on updating cluster endpoint in releases

---

## Phase 7: Testing & Refinement

### Goals
- End-to-end testing across all components
- Bug fixes and polish
- Performance optimization
- Complete documentation

### Testing Checklist

**Registration Testing:**
- [ ] User can self-register via `kubecraft register --username alice`
- [ ] Token is generated and saved to `~/.kubecraft/config`
- [ ] Duplicate username registration is rejected
- [ ] Invalid usernames are rejected (format validation)
- [ ] User limit enforced (16th user rejected)
- [ ] Registration service remains responsive under multiple requests

**Authentication Testing:**
- [ ] Token works for server management commands
- [ ] Token persists across CLI restarts (read from disk)
- [ ] Copying config file to another machine works
- [ ] Expired token shows clear error message
- [ ] RBAC properly isolates users (Alice can't access Bob's namespace)

**Functionality Testing:**
- [ ] Service created with NodePort type
- [ ] NodePort assigned is within 30000-30015 range
- [ ] Can connect with Minecraft client using OCI_IP:NODE_PORT
- [ ] Pre-flight check prevents creating server if RAM is full
- [ ] Storage persists on OCI instance disk (check /var/lib/rancher/k3s/storage)
- [ ] Namespace isolation verified via RBAC

**Load Testing:**
- [ ] Register 15 users successfully
- [ ] Create servers until RAM limit is hit
- [ ] Verify CLI accurately reports "Cluster Full" error
- [ ] Monitor CPU/Memory usage under load

### Documentation to Complete (Updates)

**README.md:**
```markdown
## Quick Start

1. Download CLI:
   ```bash
   # Linux/macOS (Intel/AMD)
   curl -L https://github.com/yourname/kubecraft/releases/latest/download/kubecraft-linux-amd64 -o kubecraft
   chmod +x kubecraft

   # Linux (ARM64, e.g., Raspberry Pi)
   curl -L https://github.com/yourname/kubecraft/releases/latest/download/kubecraft-linux-arm64 -o kubecraft
   chmod +x kubecraft
   ```

2. Register your account:
   ```bash
   kubecraft register --username yourname
   # ✅ Registration successful!
   # Configuration saved to ~/.kubecraft/config
   ```

3. Create a server:
   ```bash
   kubecraft server create myserver
   # Waiting for server to start...
   # ✅ Server Ready! Connect to: 129.146.xx.xx:30001
   ```

4. Connect using Minecraft client:
   - Server Address: `129.146.xx.xx:30001`

## Multi-Computer Setup

To use kubecraft on multiple computers, copy your configuration:

```bash
# On first computer
cat ~/.kubecraft/config

# Copy contents, then on second computer
mkdir -p ~/.kubecraft
nano ~/.kubecraft/config  # Paste contents
```

## Security Notes

- Your token is stored in `~/.kubecraft/config`
- Treat it like a password - don't share it
- Tokens expire after 5 years
- If you lose your token, contact admin for account reset
```

**ARCHITECTURE.md:**
- Update diagrams to show NodePort flow and registration service
- Explain HostPath storage strategy
- Detail the Memory Overcommitment strategy and safety checks (22GB available)
- Document authentication flow (ServiceAccount tokens)
- Explain registration service architecture and permissions
- Document Oracle Cloud Always Free Tier usage and ARM64 architecture

### Deliverables
- [ ] All functionality tested and working
- [ ] Complete documentation suite
- [ ] Clean, well-commented codebase

---

## Timeline Summary

| Week | Phase | Focus |
|------|-------|-------|
| 1 | Phase 0 | Prerequisites, environment setup, Local K3s |
| 2 | Phase 1 | Kubernetes manifests, RBAC, registration manifests |
| 3 | Phase 2 | Minecraft Docker image (multi-arch) |
| 3-4 | Phase 2.5 | Registration service (HTTP server, token generation) |
| 4-5 | Phase 3 | CLI tool with registration command and server management |
| 6 | Phase 4 | Terraform infrastructure (Oracle Cloud deployment) |
| 7 | Phase 5 | System deployment (registration service to cluster) |
| 8 | Phase 6 | CI/CD pipeline setup (multi-arch builds) |
| 9 | Phase 7 | Testing, bug fixes, documentation |

**Total Duration:** ~9 weeks (2-2.5 months) at 10-15 hours/week

**Note:** Phase 2.5 adds ~2-3 days of work but provides significant UX improvement
**Note:** Multi-arch builds add ~1 day of work but are required for Oracle Cloud ARM64

---

## Success Criteria

**MVP (Minimum Viable Product):**
- [ ] One user can self-register via CLI
- [ ] User can create 1 Minecraft server
- [ ] Server starts and is accessible via NodePort
- [ ] World data persists across stop/start
- [ ] Deployed on Oracle Cloud with Terraform (Always Free Tier)

**Complete Project:**
- [ ] All CLI commands functional with safety checks
- [ ] Self-service registration working (no admin intervention)
- [ ] RBAC properly isolating users
- [ ] Up to 1 server per user (ResourceQuota enforced)
- [ ] 5 users actively using the platform
- [ ] Registration service enforcing user limit (15 max)
- [ ] CI/CD pipeline functional (multi-arch images)
- [ ] Running at $0/month on Oracle Cloud Always Free

---

## Cost Breakdown

**Monthly Costs (Oracle Cloud Always Free Tier):**
- Ampere A1 Compute (4 OCPU, 24GB RAM): $0
- Block Storage (100GB): $0
- Data Transfer (up to 10TB): $0
- Load Balancers: $0 (using NodePort)
- **Total: $0/month** 🎉

**Comparison with AWS (Original Plan):**
| Resource | AWS Cost | Oracle Cloud |
|----------|----------|--------------|
| Compute | $60/month (t3.large: 2 vCPU, 8GB) | $0 (3 OCPU, 16GB) |
| Storage | $8/month (100GB gp3) | $0 (100GB) |
| Data Transfer | $5/month | $0 |
| **Total** | **$73/month** | **$0/month** |

**Oracle Cloud Provides More for Free:**
- 2x more RAM (16GB vs 8GB)
- 1.5x more CPU (3 OCPU vs 2 vCPU)
- 2x more storage available (200GB vs 100GB)

---

## Key Design Decisions (Revised)

### 1. Cloud Provider: Oracle Cloud vs AWS
- **Chosen:** Oracle Cloud Infrastructure (OCI)
- **Why:** Always Free Tier provides 4 OCPU + 24GB RAM + 200GB storage at $0/month
- **Trade-off:** ARM64 architecture requires multi-arch Docker builds; smaller ecosystem than AWS

### 2. Networking: NodePort vs LoadBalancer
- **Chosen:** NodePort
- **Why:** Massive cost savings ($0 vs $225/mo). Sufficient for small scale.
- **Trade-off:** Non-standard ports (e.g., :30001) for users.

### 3. Storage: HostPath (Local) vs Cloud Block Storage
- **Chosen:** Local Path (K3s default)
- **Why:** Simpler, faster, included in Always Free block storage.
- **Trade-off:** Data is tied to this specific instance (fine for single-node).

### 4. Safety: Strict Limits & Pre-flight Checks
- **Chosen:** CLI-side capacity checking
- **Why:** Prevents "noisy neighbors" from crashing the shared node via OOM (Out of Memory).

### 5. Authentication: ServiceAccount Tokens vs Client Certificates
- **Chosen:** ServiceAccount tokens (long-lived, 5 years)
- **Why:** Easier to automate in registration service. No certificate signing infrastructure needed.
- **Trade-off:** Token acts like a permanent password (must be kept secure). If lost, admin intervention required.

### 6. CPU Architecture: ARM64 (Ampere) vs x86
- **Chosen:** ARM64 (Ampere A1)
- **Why:** Only way to get Always Free compute on Oracle Cloud
- **Trade-off:** Must build multi-arch Docker images; some software may not support ARM64
- **Mitigation:** Java and K3s fully support ARM64; Docker buildx handles multi-arch builds