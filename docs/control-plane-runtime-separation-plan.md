# Kubecraft Separation Plan: Static Control Plane vs Dynamic Runtime

## Goal

Cleanly separate deployment concerns into:

- **Static bootstrap/control-plane resources** (declarative, Helm-managed)
- **Dynamic runtime resources** (created by registration service and CLI code at runtime)

Also enforce a single naming convention for cluster-scoped RBAC using `kc-*`.

---

## Ownership Model (Final State)

### Static (Helm-owned)

Installed once per cluster, upgraded declaratively:

- `Namespace`: `kubecraft-system`
- `ServiceAccount`: `registration-service`
- `ClusterRole`: `kc-registration-admin`
- `ClusterRoleBinding`: `kc-registration-admin-binding`
- `ClusterRole`: `kc-capacity-checker`
- `ClusterRoleBinding`: `kc-users-capacity-check`
- `Deployment`: `registration-service`
- `Service` (NodePort 30099): `registration-service`

### Dynamic (Code-owned)

Created and deleted at runtime by Go code:

- Per-user namespace and RBAC:
  - `Namespace`: `mc-{username}`
  - `ServiceAccount`: `{username}`
  - `Role`: `minecraft-manager`
  - `RoleBinding`: `binding-{username}`
  - `ResourceQuota`: `mc-compute-resources`
- Per-server workload resources:
  - `StatefulSet`: `{servername}`
  - `Service` (NodePort 30000-30015): `{servername}`
  - PVC from `volumeClaimTemplates`

---

## Canonical Names (Single Source of Truth: `kc-*`)

Use these names everywhere (code, chart, tests, docs):

- `kc-capacity-checker`
- `kc-users-capacity-check`
- `kc-registration-admin`
- `kc-registration-admin-binding`

> Note: this replaces the current `kubecraft-capacity-checker` constant usage.

---

## Phase 1 - Normalize Constants and RBAC Names

### 1.1 Update constants in Go

**File:** `internal/config/constants.go`

```go
// RBAC Resource Names
const (
	UserRoleName               = "minecraft-manager"
	CapacityCheckerClusterRole = "kc-capacity-checker"
	CapacityCheckerBinding     = "kc-users-capacity-check"
	RegistrationClusterRole    = "kc-registration-admin"
)
```

### 1.2 Update tests expecting old names

**File:** `internal/config/constants_test.go` (assertions)

```go
{"CapacityCheckerClusterRole", CapacityCheckerClusterRole, "kc-capacity-checker"},
{"CapacityCheckerBinding", CapacityCheckerBinding, "kc-users-capacity-check"},
```

### 1.3 Align integration workflow bootstrap RBAC

**File:** `.github/workflows/test-integration.yml`

Use:

```yaml
metadata:
  name: kc-capacity-checker
...
roleRef:
  kind: ClusterRole
  name: kc-capacity-checker
```

And:

```yaml
- name: Verify cluster is accessible
  run: |
    kubectl cluster-info
    kubectl get clusterrole kc-capacity-checker
```

---

## Phase 2 - Introduce Helm Chart for Static Control Plane

Create chart:

```bash
mkdir -p charts/kubecraft-control-plane/templates
```

### 2.1 Chart metadata

**File:** `charts/kubecraft-control-plane/Chart.yaml`

```yaml
apiVersion: v2
name: kubecraft-control-plane
description: Static Kubecraft control-plane resources
type: application
version: 0.1.0
appVersion: "1.0.0"
```

### 2.2 Centralized values

**File:** `charts/kubecraft-control-plane/values.yaml`

```yaml
namespace:
  name: kubecraft-system

registration:
  name: registration-service
  serviceAccountName: registration-service
  image:
    repository: hasanbaig786/kubecraft-registration
    tag: latest
    pullPolicy: IfNotPresent
  replicas: 1
  maxUsers: 15
  service:
    type: NodePort
    port: 8080
    targetPort: 8080
    nodePort: 30099
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 256Mi

rbac:
  capacityChecker:
    clusterRoleName: kc-capacity-checker
    bindingName: kc-users-capacity-check
  registrationAdmin:
    clusterRoleName: kc-registration-admin
    bindingName: kc-registration-admin-binding
```

### 2.3 Namespace template

**File:** `charts/kubecraft-control-plane/templates/namespace.yaml`

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: {{ .Values.namespace.name }}
  labels:
    app: kubecraft
    component: system
```

### 2.4 Capacity checker RBAC templates

**File:** `charts/kubecraft-control-plane/templates/capacity-clusterrole.yaml`

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ .Values.rbac.capacityChecker.clusterRoleName }}
  labels:
    app: kubecraft
    component: rbac
rules:
  - apiGroups: [""]
    resources: ["namespaces", "services", "pods"]
    verbs: ["get", "list"]
```

**File:** `charts/kubecraft-control-plane/templates/capacity-clusterrolebinding.yaml`

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ .Values.rbac.capacityChecker.bindingName }}
  labels:
    app: kubecraft
    component: rbac
subjects: []
roleRef:
  kind: ClusterRole
  name: {{ .Values.rbac.capacityChecker.clusterRoleName }}
  apiGroup: rbac.authorization.k8s.io
```

### 2.5 Registration RBAC templates

**File:** `charts/kubecraft-control-plane/templates/registration-clusterrole.yaml`

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ .Values.rbac.registrationAdmin.clusterRoleName }}
  labels:
    app: kubecraft
    component: registration
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["create", "get", "list"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims", "services"]
    verbs: ["get", "list", "create", "update", "delete"]
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
  - apiGroups: ["apps"]
    resources: ["statefulsets"]
    verbs: ["create", "get", "list", "patch", "update", "delete"]
  - apiGroups: [""]
    resources: ["serviceaccounts"]
    verbs: ["create", "get"]
  - apiGroups: ["rbac.authorization.k8s.io"]
    resources: ["roles"]
    verbs: ["create", "get", "escalate"]
  - apiGroups: ["rbac.authorization.k8s.io"]
    resources: ["rolebindings"]
    verbs: ["create", "get", "bind"]
  - apiGroups: ["rbac.authorization.k8s.io"]
    resources: ["clusterrolebindings"]
    verbs: ["get", "update", "patch"]
    resourceNames: ["{{ .Values.rbac.capacityChecker.bindingName }}"]
  - apiGroups: [""]
    resources: ["resourcequotas"]
    verbs: ["create", "get"]
  - apiGroups: [""]
    resources: ["serviceaccounts/token"]
    verbs: ["create"]
```

**File:** `charts/kubecraft-control-plane/templates/registration-clusterrolebinding.yaml`

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ .Values.rbac.registrationAdmin.bindingName }}
  labels:
    app: kubecraft
    component: registration
subjects:
  - kind: ServiceAccount
    name: {{ .Values.registration.serviceAccountName }}
    namespace: {{ .Values.namespace.name }}
roleRef:
  kind: ClusterRole
  name: {{ .Values.rbac.registrationAdmin.clusterRoleName }}
  apiGroup: rbac.authorization.k8s.io
```

### 2.6 Registration workload templates

**File:** `charts/kubecraft-control-plane/templates/registration-serviceaccount.yaml`

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .Values.registration.serviceAccountName }}
  namespace: {{ .Values.namespace.name }}
  labels:
    app: kubecraft
    component: registration
```

**File:** `charts/kubecraft-control-plane/templates/registration-deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Values.registration.name }}
  namespace: {{ .Values.namespace.name }}
  labels:
    app: kubecraft
    component: registration
spec:
  replicas: {{ .Values.registration.replicas }}
  selector:
    matchLabels:
      app: kubecraft
      component: registration
  template:
    metadata:
      labels:
        app: kubecraft
        component: registration
    spec:
      serviceAccountName: {{ .Values.registration.serviceAccountName }}
      containers:
        - name: registration
          image: "{{ .Values.registration.image.repository }}:{{ .Values.registration.image.tag }}"
          imagePullPolicy: {{ .Values.registration.image.pullPolicy }}
          ports:
            - containerPort: 8080
          env:
            - name: MAX_USERS
              value: "{{ .Values.registration.maxUsers }}"
          resources:
            requests:
              cpu: {{ .Values.registration.resources.requests.cpu }}
              memory: {{ .Values.registration.resources.requests.memory }}
            limits:
              cpu: {{ .Values.registration.resources.limits.cpu }}
              memory: {{ .Values.registration.resources.limits.memory }}
```

**File:** `charts/kubecraft-control-plane/templates/registration-service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ .Values.registration.name }}
  namespace: {{ .Values.namespace.name }}
  labels:
    app: kubecraft
    component: registration
spec:
  type: {{ .Values.registration.service.type }}
  selector:
    app: kubecraft
    component: registration
  ports:
    - port: {{ .Values.registration.service.port }}
      targetPort: {{ .Values.registration.service.targetPort }}
      nodePort: {{ .Values.registration.service.nodePort }}
      protocol: TCP
```

---

## Phase 3 - Replace Bootstrap Flow with Helm

### 3.1 Update Makefile target

**File:** `Makefile`

Replace `cluster-setup` with:

```make
cluster-setup:
	helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane
```

(For local dev namespace handling parity, the chart itself creates `kubecraft-system`.)

### 3.2 Replace manual apply docs/commands

Old:

```bash
kubectl apply -f manifests/system-templates/
kubectl apply -f manifests/registration-templates/
```

New:

```bash
helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane
```

---

## Phase 4 - Decommission Dynamic YAML Templates and Legacy Scripts

### 4.1 Remove dynamic manifest templates from deployment surface

No longer used as primary mechanism:

- `manifests/user-templates/*`
- `manifests/server-templates/*`

These resources remain code-managed by:

- `internal/registration/handler.go`
- `internal/k8s/namespace.go`
- `internal/k8s/rbac.go`
- `internal/k8s/server.go`

### 4.2 Retire script paths that mutate dynamic resources via templating

Deprecate:

- `scripts/create-user.sh`
- `scripts/delete-user.sh`

Preferred runtime interfaces:

- `kubecraft register --username <name>`
- `kubecraft server create|list|start|stop|delete ...`

---

## Phase 5 - Testing Split by Concern

### 5.1 Static control-plane tests (Helm + manifest rendering)

Add CI steps:

```bash
helm lint ./charts/kubecraft-control-plane
helm template kubecraft-control-plane ./charts/kubecraft-control-plane | kubectl apply --dry-run=client -f -
```

### 5.2 Dynamic runtime tests (Go integration + RBAC behavior)

Keep/extend integration tests:

```bash
go test -v -race -tags=integration ./internal/...
```

Use control-plane install before dynamic tests:

```bash
helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane
```

---

## Phase 6 - Documentation and Operational Guardrails

### 6.1 Update docs

Update these files to reflect ownership boundaries:

- `README.md`
- `AGENTS.md`
- `CLAUDE.md`
- `TESTING.md`

### 6.2 Add boundary statement (recommended text)

```md
Terraform owns infrastructure lifecycle.
Helm owns static control-plane Kubernetes resources.
Kubecraft services/CLI own dynamic tenant and server runtime resources.
```

### 6.3 Optional startup validation

At registration service startup, verify the following exist and fail fast if missing:

- `ClusterRole` `kc-capacity-checker`
- `ClusterRoleBinding` `kc-users-capacity-check`
- `ClusterRole` `kc-registration-admin`
- `ClusterRoleBinding` `kc-registration-admin-binding`

---

## Execution Order (Strict)

1. Normalize `kc-*` constants and tests.
2. Add Helm chart and templates.
3. Switch `cluster-setup` and docs to Helm install.
4. Update CI to lint/render chart + install chart before runtime integration tests.
5. Deprecate dynamic manifest/script paths from operational docs.
6. Remove deprecated files only after CI passes in two consecutive runs.

---

## Definition of Done

- All cluster-scoped and registration bootstrap resources are deployed only via Helm chart.
- Dynamic user/server resources are created only via Go runtime code paths.
- No remaining references to `kubecraft-capacity-checker`; all use `kc-capacity-checker`.
- CI green for:
  - unit tests
  - integration tests
  - Helm lint/render validation
- Docs reflect final ownership model and bootstrap commands.
