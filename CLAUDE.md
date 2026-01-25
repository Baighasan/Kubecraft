# Minecraft Server Platform - Project Plan

## Project Overview

Self-service Minecraft server hosting platform where users can create, manage, and connect to their own Minecraft servers through a CLI tool. Built with Kubernetes, Terraform, and Go to demonstrate DevOps and platform engineering skills.

## Architecture Summary

**Single-Node Kubernetes Setup:**
- One EC2 instance (t3.large: 2 vCPU, 8GB RAM) running K3s
- All components run as pods on this single node
- Terraform provisions AWS infrastructure
- K3s manages container orchestration
- **Networking:** `NodePort` services exposing ports 30000-30015 directly on the host IP
- **Storage:** `local-path` StorageClass writing directly to the EC2 NVMe disk (HostPath)

**Core Components:**
1. **CLI Tool**: Go-based command-line interface with direct Kubernetes API access and **pre-flight capacity checks**
2. **Registration Service**: HTTP service for self-service user account creation (NodePort 30099)
3. **User Namespaces**: One namespace per user with RBAC-enforced isolation
4. **Minecraft Pods**: StatefulSets within user namespaces (up to 1 per user)
5. **System Namespace**: Registration service and optional monitoring tools

**Tech Stack:**
- Infrastructure: AWS (EC2, VPC), Terraform, K3s
- Application Code: Go 1.25.5 with monorepo structure (single module, multiple binaries)
- CLI Tool: Go with Cobra framework, client-go (Kubernetes client library)
- Registration Service: Go HTTP server, client-go with elevated permissions
- Container Orchestration: K3s (lightweight Kubernetes)
- Authentication: Kubernetes RBAC + ServiceAccount tokens (5-year expiration)
- CI/CD: GitHub Actions
- Container Registry: Docker Hub

> **📚 For detailed code structure and package responsibilities, see [Code Structure & Architecture](#code-structure--architecture) section below.**

## Project Scope

**Users:** 5 people
**Servers per user:** Up to 1
**Concurrent servers:** 2-3 running simultaneously (strict memory limits applied)
**Total servers:** Up to 15 (most stopped to save resources)
**Monthly cost:** ~$73 (single t3.large EC2 instance with 100GB storage)

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
├── terraform/                                # AWS infrastructure as code
│   ├── main.tf
│   ├── compute.tf
│   ├── security.tf
│   └── variables.tf
│
├── scripts/                                  # Admin helper scripts
│   └── delete-user.sh                        # Manual user cleanup
│
└── .github/workflows/                        # CI/CD pipelines
    ├── minecraft-image.yml
    ├── registration-image.yml
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
- **Runs on:** EC2 instance as a pod in `kubecraft-system` namespace
- **Exposes:** NodePort 30099 for HTTP endpoint
- **Permissions:** Elevated (ClusterRole) to create namespaces and RBAC
- **Dependencies:** Imports `internal/k8s`, `internal/registration`, `internal/config`

**`internal/k8s/`** - Kubernetes Operations (Shared Library)
- **Purpose:** Wrapper around client-go for common K8s operations
- **Used by:** Both CLI and registration service
- **Provides:**
  - `client.go` - Initialize Kubernetes clientset (InClusterConfig or kubeconfig)
  - `namespace.go` - Create/check namespaces, count users
  - `rbac.go` - Create ServiceAccounts, Roles, RoleBindings, patch ClusterRoleBindings
  - `token.go` - Generate ServiceAccount tokens via TokenRequest API
  - `server.go` - Create/delete StatefulSets, allocate NodePorts, check capacity

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
  - `constants.go` - MAX_USERS (15), cluster endpoint, NodePort range (30000-30015)

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
# Build CLI tool
go build -o bin/kubecraft ./cmd/kubecraft

# Build registration service
go build -o bin/registration-server ./cmd/registration-server

# Run locally (development)
go run ./cmd/kubecraft register --username alice
go run ./cmd/registration-server

# Add/update dependencies
go mod tidy
```

### Data Flow Architecture

**Registration Flow (One-Time):**
```
User's Computer                    EC2 Instance (Kubernetes Cluster)
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
User's Computer                    EC2 Instance (Kubernetes Cluster)
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
- **Shift Left:** Develop and test entirely on local clusters before deploying to AWS
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

**Kubernetes concepts (in-depth):**
- Pods, StatefulSets, Services, Namespaces
- **NodePort Networking** vs LoadBalancers
- RBAC (Roles, RoleBindings)
- PersistentVolumeClaims (local-path)
- ResourceQuotas & Limits
- Labels and selectors

**AWS basics:**
- EC2 instances and instance types
- VPC, subnets, and networking
- **Security Groups (Ingress ranges)**
- Elastic IPs
- Free Tier limits and pricing

**Go basics:**
- Syntax and idioms
- Package management (go mod)
- Building binaries
- Working with structs and interfaces
- Error handling patterns
- Goroutines and channels (basic)

### Deliverables
- [ ] Development environment configured (Docker, kubectl, Terraform, Go installed)
- [ ] AWS, GitHub, Docker Hub accounts created
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
  - CPU: 1500m request, 2250m limit
  - Memory: 1536Mi request, 3Gi limit
  - PVCs: max 1 per namespace
- **role.yaml** - `minecraft-manager` Role granting permissions for:
  - PVCs and Services (create, delete, get, list)
  - StatefulSets (create, get, list, patch, update, delete)
  - Pods and pod logs (read-only)
- **rolebinding.yaml** - Binds the Role to the user's ServiceAccount

**📁 manifests/server-templates/** - Minecraft server resources (created by CLI)
- **statefulset.yaml** - Minecraft server StatefulSet with:
  - Container image: `hasanbaig786/kubecraft`
  - Resource requests: 768Mi RAM, 500m CPU
  - Resource limits: 1Gi RAM, 750m CPU
  - Volume mount: `/data` (for world persistence)
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
t3.large total: 8GB RAM, 2 vCPU
System overhead (K3s, OS, monitoring): ~2GB RAM, ~500m CPU
Available for workloads: ~6GB RAM, ~1.5 vCPU

Per-server resources:
  requests: 768Mi RAM, 500m CPU
  limits: 1Gi RAM, 750m CPU

Capacity calculation:
  Average case (2 servers running): 2 × 768Mi = 1.5GB RAM
  Max case (3 servers running): 3 × 1Gi = 3GB RAM
  Total cluster capacity: 5 users × 3GB = 15GB potential, but only ~4-5 servers run concurrently

Safety: Pre-flight check prevents server creation if available RAM < 1.5GB
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
- Configure for Kubernetes deployment
- Test locally

### Dockerfile Structure

Standard structure as previously defined, ensuring Volume mount points align with PVC.

### What to Learn

**Dockerfile Best Practices:**
- Multi-stage builds
- Layer caching

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

### Deliverables
- [ ] Working Dockerfile for Minecraft server
- [ ] Startup script with environment variable configuration
- [ ] Image built and tested locally
- [ ] Image pushed to Docker Hub
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
- [x] All `internal/k8s/` files implemented (client, namespace, rbac, token)
- [x] All `internal/registration/` files implemented (validator, handler)
- [x] All `internal/config/` files implemented (constants)
- [x] `cmd/registration-server/main.go` HTTP server running
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
   - `CreateServer()` - Create PVC, StatefulSet, Service
   - `DeleteServer()` - Remove all server resources
   - `ListServers()` - Get all servers in namespace
   - `ScaleServer()` - Start/stop by scaling replicas
   - `WaitForReady()` - Poll until pod is running and ready

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
- Compare against available RAM (8GB total - 2GB system overhead)
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
- [ ] `kubecraft register` command implemented
- [ ] Cluster endpoint hardcoded in binary (no --cluster flag needed)
- [ ] CLI sends HTTP request to registration service
- [ ] CLI saves received token to `~/.kubecraft/config`
- [ ] CLI reads token from config for all server commands
- [ ] CLI tool implements "Pre-flight" memory check
- [ ] `create` command waits for Pod to be Ready
- [ ] `create` command returns the specific NodePort (e.g., 30001) to the user
- [ ] Optional: `config export/import` commands for multi-computer usage
- [ ] Comprehensive help text and table formatting

---

## Phase 4: Terraform Infrastructure

### Goals
- Define AWS infrastructure as code
- Provision EC2 instance with K3s
- Configure Security Groups for NodePort Range
- Provision adequate storage for Local Path

### Main Configuration Files Changes

**variables.tf:**
```hcl
variable "node_port_range_start" {
  description = "Start of NodePort range"
  type        = number
  default     = 30000
}

variable "node_port_range_end" {
  description = "End of NodePort range"
  type        = number
  default     = 30015 # Accommodates 15 servers
}
```

**security.tf (UPDATED):**
```hcl
resource "aws_security_group" "k3s" {
  name        = "${var.cluster_name}-sg"
  description = "Security group for K3s cluster"
  vpc_id      = aws_vpc.main.id

  # SSH access
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.my_ip]
    description = "SSH from my IP"
  }

  # Kubernetes API
  ingress {
    from_port   = 6443
    to_port     = 6443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Kubernetes API"
  }

  # UPDATED: Minecraft NodePort Range
  ingress {
    from_port   = var.node_port_range_start
    to_port     = var.node_port_range_end
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Minecraft NodePorts (30000-30015)"
  }

  # Allow all outbound
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.cluster_name}-sg"
  }
}
```

**compute.tf (UPDATED):**
```hcl
resource "aws_instance" "k3s" {
  # ... (other config same) ...

  root_block_device {
    volume_size = 100 # INCREASED: To accommodate 15 servers x 5GB + OS
    volume_type = "gp3"
  }

  # ... (user_data same) ...
}
```

### What to Learn

**Terraform AWS Provider:**
- Volume Sizing (100GB buffer)
- Security Group Ranges

**AWS Networking:**
- VPC, Subnets, Route Tables

**K3s Installation:**
- User Data scripts

### Deliverables
- [ ] Security Group opens ports 30000-30015 (includes 30099 for registration)
- [ ] EC2 instance provisioned with 100GB GP3 storage
- [ ] Admin kubeconfig retrievable via SCP
- [ ] Can connect to cluster with kubectl

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
4. Verify registration service is accessible at `http://EC2_IP:30099`

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
- [ ] Registration service deployed and running
- [ ] Registration service accessible via NodePort 30099
- [ ] Can successfully register users via CLI
- [ ] Tokens generated are valid and work for server management
- [ ] User limit enforcement working (rejects 16th user)
- [ ] Admin deletion script for cleanup
- [ ] Documentation for admin deployment process

---

## Phase 6: CI/CD Pipeline

### Goals
- Automate building and pushing Docker images
- Automate CLI binary builds and releases
- Ensure cluster endpoint is updated in CLI on deployment

### GitHub Actions Workflows

**Existing:**
- `minecraft-image.yml` - Builds and pushes Minecraft server image
- `cli-release.yml` - Builds CLI binaries for multiple platforms
- `test.yml` - Runs unit and integration tests

**New:**
- `registration-image.yml` - Builds and pushes registration service image

**Build-time Configuration:**
- CLI build embeds current cluster endpoint (update when EC2 IP changes)
- Version tagging for releases

### Deliverables
- [ ] Working CI/CD pipeline for Minecraft image
- [ ] Working CI/CD pipeline for registration service image
- [ ] Automated CLI builds for multiple platforms (Linux, macOS, Windows)
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
- [ ] Can connect with Minecraft client using EC2_IP:NODE_PORT
- [ ] Pre-flight check prevents creating server if RAM is full
- [ ] Storage persists on EC2 host disk (check /var/lib/rancher/k3s/storage)
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
   # Linux/macOS
   curl -L https://github.com/yourname/kubecraft/releases/latest/download/kubecraft -o kubecraft
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
   # ✅ Server Ready! Connect to: 54.123.45.67:30001
   ```

4. Connect using Minecraft client:
   - Server Address: `54.123.45.67:30001`

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
- Detail the Memory Overcommitment strategy and safety checks
- Document authentication flow (ServiceAccount tokens)
- Explain registration service architecture and permissions

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
| 3 | Phase 2 | Minecraft Docker image |
| 3-4 | Phase 2.5 | Registration service (HTTP server, token generation) |
| 4-5 | Phase 3 | CLI tool with registration command and server management |
| 6 | Phase 4 | Terraform infrastructure (AWS deployment) |
| 7 | Phase 5 | System deployment (registration service to cluster) |
| 8 | Phase 6 | CI/CD pipeline setup |
| 9 | Phase 7 | Testing, bug fixes, documentation |

**Total Duration:** ~9 weeks (2-2.5 months) at 10-15 hours/week

**Note:** Phase 2.5 adds ~2-3 days of work but provides significant UX improvement

---

## Success Criteria

**MVP (Minimum Viable Product):**
- [ ] One user can self-register via CLI
- [ ] User can create 1 Minecraft server
- [ ] Server starts and is accessible via NodePort
- [ ] World data persists across stop/start
- [ ] Deployed on AWS with Terraform

**Complete Project:**
- [ ] All CLI commands functional with safety checks
- [ ] Self-service registration working (no admin intervention)
- [ ] RBAC properly isolating users
- [ ] Up to 3 servers per user (ResourceQuota enforced)
- [ ] 5 users actively using the platform
- [ ] Registration service enforcing user limit (15 max)
- [ ] CI/CD pipeline functional

---

## Cost Breakdown (Revised)

**Monthly Costs:**
- EC2 t3.large: ~$60/month
- EBS Storage (100GB gp3): ~$8/month
- Data Transfer: ~$5/month
- Load Balancers: $0 (Removed)
- **Total: ~$73/month** (~$15/user for 5 users)

**Cost Optimization:**
- Use Spot Instance (60-70% cheaper, with interruption risk)
- Implement idle server auto-shutdown

---

## Key Design Decisions (Revised)

### 1. Networking: NodePort vs LoadBalancer
- **Chosen:** NodePort
- **Why:** Massive cost savings ($0 vs $225/mo). Sufficient for small scale.
- **Trade-off:** Non-standard ports (e.g., :30001) for users.

### 2. Storage: HostPath (Local) vs EBS
- **Chosen:** Local Path (K3s default)
- **Why:** Simpler, faster, no AWS volume attachment limits (max ~25 attachments per instance).
- **Trade-off:** Data is tied to this specific EC2 instance (fine for single-node).

### 3. Safety: Strict Limits & Pre-flight Checks
- **Chosen:** CLI-side capacity checking
- **Why:** Prevents "noisy neighbors" from crashing the shared node via OOM (Out of Memory).

### 4. Authentication: ServiceAccount Tokens vs Client Certificates
- **Chosen:** ServiceAccount tokens (long-lived, 5 years)
- **Why:** Easier to automate in registration service. No certificate signing infrastructure needed.
- **Trade-off:** Token acts like a permanent password (must be kept secure). If lost, admin intervention required.