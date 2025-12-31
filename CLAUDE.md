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
- CLI Tool: Go, client-go (Kubernetes client library)
- Container Orchestration: K3s (lightweight Kubernetes)
- Authentication: Kubernetes RBAC + ServiceAccount tokens
- Registration: Go HTTP service for self-service user onboarding
- CI/CD: GitHub Actions
- Container Registry: Docker Hub

## Project Scope

**Users:** 5 people
**Servers per user:** Up to 1
**Concurrent servers:** 2-3 running simultaneously (strict memory limits applied)
**Total servers:** Up to 15 (most stopped to save resources)
**Monthly cost:** ~$73 (single t3.large EC2 instance with 100GB storage)

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

### Manifests to Create

**User Namespace Template (per user):**
```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: mc-{username}
  labels:
    app: kubecraft
    user: {username}

---
# resourcequota.yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-resources
  namespace: mc-{username}
spec:
  hard:
    requests.cpu: "1500m"      # Reduced: allows 2 servers avg per user (2 × 500m)
    requests.memory: 1536Mi    # Reduced: allows 2 servers avg per user (2 × 768Mi)
    limits.cpu: "2250m"        # Reduced: allows up to 3 servers max (3 × 750m)
    limits.memory: 3Gi         # Reduced: allows up to 3 servers max (3 × 1Gi)
    persistentvolumeclaims: "1"

# CAPACITY PLANNING NOTE:
# t3.large total: 8GB RAM, 2 vCPU
# System overhead (K3s, OS, monitoring): ~2GB RAM, ~500m CPU
# Available for workloads: ~6GB RAM, ~1.5 vCPU
#
# Per-server resources:
#   requests: 768Mi RAM, 500m CPU
#   limits: 1Gi RAM, 750m CPU
#
# Capacity calculation:
#   Average case (2 servers running): 2 × 768Mi = 1.5GB RAM
#   Max case (3 servers running): 3 × 1Gi = 3GB RAM
#   Total cluster capacity: 5 users × 3GB = 15GB potential, but only ~4-5 servers run concurrently
#
# Safety: Pre-flight check prevents server creation if available RAM < 1.5GB

---
# role.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: minecraft-manager
  namespace: mc-{username}
rules:
- apiGroups: [""]
  resources: ["persistentvolumeclaims", "services"]
  verbs: ["get", "list", "create", "update", "delete"]
- apiGroups: ["apps"]
  resources: ["statefulsets"]
  verbs: ["get", "list", "create", "update", "delete", "patch"]
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list"]
# Note: metrics.k8s.io removed - capacity checks use pod.Spec.Resources instead

---
# rolebinding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {username}-binding
  namespace: mc-{username}
subjects:
- kind: ServiceAccount
  name: {username}
  namespace: mc-{username}
roleRef:
  kind: Role
  name: minecraft-manager
  apiGroup: rbac.authorization.k8s.io

---
# serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {username}
  namespace: mc-{username}

---
# clusterrole.yaml (Applied once by admin, shared by all users)
# This grants read-only access to namespaces and services for capacity/port checks
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubecraft-capacity-checker
rules:
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list"]
- apiGroups: [""]
  resources: ["services", "pods"]
  verbs: ["get", "list"]
  # Users can only list across mc-* namespaces (enforced by label selector in code)

---
# clusterrolebinding.yaml (Applied once by admin)
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubecraft-users-capacity-check
subjects:
# Add each user's ServiceAccount here during onboarding
- kind: ServiceAccount
  name: {username}
  namespace: mc-{username}
roleRef:
  kind: ClusterRole
  name: kubecraft-capacity-checker
  apiGroup: rbac.authorization.k8s.io
```

**Minecraft Server Template (within user namespace):**
```yaml
# statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {servername}
  namespace: mc-{username}
  labels:
    app: minecraft
    server: {servername}
    owner: {username}
spec:
  serviceName: {servername}
  replicas: 1
  selector:
    matchLabels:
      app: minecraft
      server: {servername}
  template:
    metadata:
      labels:
        app: minecraft
        server: {servername}
    spec:
      containers:
      - name: minecraft
        image: your-dockerhub/minecraft:latest
        env:
        - name: VERSION
          value: "1.20.1"
        - name: GAME_MODE
          value: "survival"
        - name: MAX_PLAYERS
          value: "20"
        ports:
        - containerPort: 25565
          protocol: TCP
        resources:
          # STRICT LIMITS ENFORCED to prevent Node OOM
          requests:
            memory: "768Mi"  # Reduced: realistic for Minecraft (768MB)
            cpu: "500m"
          limits:
            memory: "1Gi"    # Reduced: prevents memory overcommitment
            cpu: "750m"      # Reduced: allows more concurrent servers
        volumeMounts:
        - name: data
          mountPath: /data
        readinessProbe:
          tcpSocket:
            port: 25565
          initialDelaySeconds: 30
          periodSeconds: 10
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: "local-path" # Writes to host disk
      resources:
        requests:
          storage: 5Gi

---
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: {servername}
  namespace: mc-{username}
spec:
  type: NodePort # CHANGED: No LoadBalancer costs
  selector:
    app: minecraft
    server: {servername}
  ports:
  - port: 25565
    targetPort: 25565
    # nodePort will be assigned automatically (30000-32767) 
    # or manually assigned by CLI logic
    protocol: TCP
```

**System Namespace (Optional):**
```yaml
# idle-monitor-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: idle-monitor
  namespace: system
spec:
  schedule: "*/30 * * * *"  # Every 30 minutes
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: idle-monitor
          containers:
          - name: monitor
            image: your-dockerhub/idle-monitor:latest
            env:
            - name: IDLE_THRESHOLD_MINUTES
              value: "60"
          restartPolicy: OnFailure
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

### Deliverables
- [ ] Registration service implementation (`cmd/registration-server/`, `pkg/registration/`)
- [ ] Dockerfile for registration service
- [ ] Service can create all user resources (namespace, RBAC, etc.)
- [ ] Service generates valid ServiceAccount tokens
- [ ] Username validation (format, uniqueness, reserved names)
- [ ] User limit enforcement (max 15 users)
- [ ] Tested locally with curl and in k3d cluster
- [ ] Image pushed to Docker Hub

---

## Phase 3: CLI Tool with Kubernetes Client

### Goals
- Build Go-based CLI tool with direct Kubernetes API access
- Implement self-service registration command
- Implement Pre-flight Capacity Checks (Safety Mechanism)
- Implement Polling Logic for NodePort retrieval
- Embed cluster endpoint in CLI binary

### Project Structure

```
cmd/cli/
  main.go           # CLI entrypoint
  config.go         # Hardcoded cluster endpoint constants
  register.go       # Self-service registration command
  server.go         # Server management commands (create, list, stop, delete)

pkg/k8s/
  client.go         # Kubernetes client wrapper
  server.go         # Server operations (create, delete, capacity checks)
  auth.go           # Token and kubeconfig management
```

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

### Key Code Examples

**Server Creation (pkg/k8s/server.go) - UPDATED:**
```go
package k8s

import (
    "context"
    "fmt"
    "time"
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/intstr"
)

// CheckNodeCapacity ensures the node has sufficient RAM before scheduling a new server
// This prevents OOM (Out of Memory) situations that would crash the entire node
func (c *Client) CheckNodeCapacity(ctx context.Context) error {
    const (
        totalNodeRAM     = 8 * 1024 * 1024 * 1024  // 8GB in bytes (t3.large)
        systemOverhead   = 2 * 1024 * 1024 * 1024  // 2GB reserved for K3s, OS, etc.
        safetyMargin     = 1536 * 1024 * 1024      // 1.5GB safety buffer
        newServerRequest = 768 * 1024 * 1024       // 768Mi per new server (from requests.memory)
    )

    availableRAM := totalNodeRAM - systemOverhead

    // Strategy: Sum all current Pod memory requests across the cluster
    // This is more reliable than metrics-server (which may not be installed)
    // and reflects Kubernetes' scheduling decisions

    // 1. Get all namespaces with kubecraft label
    namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
        LabelSelector: "app=kubecraft",
    })
    if err != nil {
        return fmt.Errorf("failed to list namespaces: %w", err)
    }

    // 2. Sum memory requests from all running pods
    var totalMemoryRequests int64 = 0

    for _, ns := range namespaces.Items {
        pods, err := c.clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{
            LabelSelector: "app=minecraft",
            FieldSelector: "status.phase=Running", // Only count running pods
        })
        if err != nil {
            return fmt.Errorf("failed to list pods in namespace %s: %w", ns.Name, err)
        }

        for _, pod := range pods.Items {
            for _, container := range pod.Spec.Containers {
                // Extract memory request
                if memRequest, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
                    totalMemoryRequests += memRequest.Value()
                }
            }
        }
    }

    // 3. Calculate free RAM
    freeRAM := availableRAM - totalMemoryRequests

    // 4. Check if we have enough space for the new server + safety margin
    requiredRAM := newServerRequest + safetyMargin

    if freeRAM < requiredRAM {
        // Build helpful error message
        return fmt.Errorf(
            "insufficient cluster capacity\n"+
                "  Available RAM: %.2f GB\n"+
                "  Currently Used: %.2f GB\n"+
                "  Free RAM: %.2f GB\n"+
                "  Required (new server + safety margin): %.2f GB\n\n"+
                "Try:\n"+
                "  • Stop an idle server: kubecraft server stop <name>\n"+
                "  • Delete unused servers: kubecraft server delete <name>\n"+
                "  • List your servers: kubecraft server list",
            bytesToGB(availableRAM),
            bytesToGB(totalMemoryRequests),
            bytesToGB(freeRAM),
            bytesToGB(requiredRAM),
        )
    }

    return nil // Capacity check passed
}

// bytesToGB converts bytes to gigabytes for human-readable output
func bytesToGB(bytes int64) float64 {
    return float64(bytes) / (1024 * 1024 * 1024)
}

func (c *Client) CreateServer(ctx context.Context, config ServerConfig) error {
    
    // 1. Run Pre-flight Check
    if err := c.CheckNodeCapacity(ctx); err != nil {
        return fmt.Errorf("capacity check failed: %w", err)
    }

    // 2. Create PVC (standard)
    // ... (PVC creation code, ensure storageClassName is "local-path") ...
    
    // 3. Create StatefulSet (standard)
    // ... (StatefulSet creation code) ...
    
    // 4. Allocate NodePort (CRITICAL: prevents port collisions)
    nodePort, err := c.allocateNodePort(ctx)
    if err != nil {
        return fmt.Errorf("failed to allocate NodePort: %w", err)
    }

    // 5. Create Service (NodePort)
    service := &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name:      config.Name,
            Namespace: c.namespace,
            Labels: map[string]string{
                "app":    "minecraft",
                "server": config.Name,
            },
        },
        Spec: corev1.ServiceSpec{
            Type: corev1.ServiceTypeNodePort, // CHANGED from LoadBalancer
            Selector: map[string]string{
                "app":    "minecraft",
                "server": config.Name,
            },
            Ports: []corev1.ServicePort{
                {
                    Port:       25565,
                    TargetPort: intstr.FromInt(25565),
                    Protocol:   corev1.ProtocolTCP,
                    NodePort:   nodePort, // EXPLICIT assignment in our range
                },
            },
        },
    }

    _, err = c.clientset.CoreV1().Services(c.namespace).Create(ctx, service, metav1.CreateOptions{})
    return err
}

// allocateNodePort finds the first available port in the reserved range (30000-30015)
func (c *Client) allocateNodePort(ctx context.Context) (int32, error) {
    const (
        minPort = 30000
        maxPort = 30015 // Supports up to 16 servers (more than our 15-server limit)
    )

    // 1. List all Services across all mc-* namespaces to find used ports
    usedPorts := make(map[int32]bool)

    // Get all namespaces with label app=kubecraft
    namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
        LabelSelector: "app=kubecraft",
    })
    if err != nil {
        return 0, fmt.Errorf("failed to list namespaces: %w", err)
    }

    // Collect all NodePorts in use across all user namespaces
    for _, ns := range namespaces.Items {
        services, err := c.clientset.CoreV1().Services(ns.Name).List(ctx, metav1.ListOptions{
            LabelSelector: "app=minecraft",
        })
        if err != nil {
            return 0, fmt.Errorf("failed to list services in namespace %s: %w", ns.Name, err)
        }

        for _, svc := range services.Items {
            for _, port := range svc.Spec.Ports {
                if port.NodePort != 0 {
                    usedPorts[port.NodePort] = true
                }
            }
        }
    }

    // 2. Find the first available port in our range
    for port := minPort; port <= maxPort; port++ {
        if !usedPorts[int32(port)] {
            return int32(port), nil
        }
    }

    // 3. No ports available - cluster is at maximum capacity
    return 0, fmt.Errorf("all NodePorts in range %d-%d are allocated (max %d servers reached)",
        minPort, maxPort, maxPort-minPort+1)
}

// WaitForReady polls until the pod is running and returns the NodePort
func (c *Client) WaitForReady(ctx context.Context, name string) (int32, error) {
    // Poll for up to 5 minutes (typical Minecraft startup time)
    timeout := time.After(5 * time.Minute)
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-timeout:
            return 0, fmt.Errorf("timeout waiting for server %s to become ready", name)

        case <-ticker.C:
            // 1. Check if Pod is Running and Ready
            pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{
                LabelSelector: fmt.Sprintf("app=minecraft,server=%s", name),
            })
            if err != nil {
                continue // Retry on transient errors
            }

            if len(pods.Items) == 0 {
                continue // Pod not created yet
            }

            pod := pods.Items[0]

            // Check if pod is Running
            if pod.Status.Phase != corev1.PodRunning {
                continue
            }

            // Check if readiness probe has passed
            ready := false
            for _, condition := range pod.Status.Conditions {
                if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
                    ready = true
                    break
                }
            }

            if !ready {
                continue
            }

            // 2. Pod is ready! Retrieve the NodePort from Service
            svc, err := c.clientset.CoreV1().Services(c.namespace).Get(ctx, name, metav1.GetOptions{})
            if err != nil {
                return 0, fmt.Errorf("server is ready but failed to get service: %w", err)
            }

            if len(svc.Spec.Ports) == 0 {
                return 0, fmt.Errorf("service has no ports configured")
            }

            nodePort := svc.Spec.Ports[0].NodePort
            if nodePort == 0 {
                return 0, fmt.Errorf("service NodePort not assigned")
            }

            return nodePort, nil
        }
    }
}
```

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