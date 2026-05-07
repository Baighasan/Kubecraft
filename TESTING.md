# Kubecraft Testing Guide

This document explains how to run tests for the Kubecraft project.

**Ownership boundary:**
- Helm owns static control-plane Kubernetes resources.
- Go code owns dynamic tenant and server runtime resources.
- Tests must reflect this split: validate Helm charts for static resources, and use Go integration tests for dynamic behavior.

## Test Structure

```
charts/kubecraft-control-plane/
├── templates/               # Static control-plane Kubernetes resources
├── values.yaml              # Tunable configuration
└── Chart.yaml               # Chart metadata

scripts/
└── test-all.sh              # Helm lint + Go integration tests

.github/workflows/
├── test-unit.yml            # Unit tests (no cluster needed)
├── test-integration.yml     # Integration tests with k3d cluster (installs Helm chart)
└── test-manifests.yml       # Helm lint + template dry-run
```

## Quick Start

### Run All Tests (Recommended)

```bash
./scripts/test-all.sh
```

This will run:
1. Helm chart validation (lint + render)
2. Go integration tests

### Run Individual Test Suites

**Helm chart validation:**
```bash
helm lint ./charts/kubecraft-control-plane
helm template kubecraft-control-plane ./charts/kubecraft-control-plane | kubectl apply --dry-run=client -f -
```

**Go integration tests:**
```bash
go test -p 1 -tags=integration ./internal/...
```

> **Note:** Integration tests currently run with `-p 1` (serial package execution) as a temporary mitigation to avoid concurrent `ClusterRoleBinding` mutation conflicts. This will be removed once conflict-safe update logic is implemented.

## Prerequisites

### Local Testing

1. **Kubernetes cluster** - One of:
   - k3d (recommended): `k3d cluster create test-cluster --agents 1`
   - minikube: `minikube start`
   - kind: `kind create cluster`

2. **kubectl** installed and configured

3. **Cluster must be running** before tests

### Setting Up Test Cluster

**Using k3d (recommended):**
```bash
# Install k3d (if not installed)
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# Create test cluster
k3d cluster create kubecraft-test --agents 1

# Install static control-plane resources
helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane

# Verify cluster is ready
kubectl get nodes
```

**Using minikube:**
```bash
# Start cluster
minikube start

# Verify cluster is ready
kubectl get nodes
```

## Test Descriptions

### 1. Helm Chart Validation

**What it tests:**
- ✅ Helm chart syntax (`helm lint`)
- ✅ Kubernetes API compliance (using `helm template | kubectl apply --dry-run=client`)
- ✅ Template rendering correctness

**Duration:** ~30 seconds

**Run:**
```bash
helm lint ./charts/kubecraft-control-plane
helm template kubecraft-control-plane ./charts/kubecraft-control-plane | kubectl apply --dry-run=client -f -
```

### 2. Go Integration Tests

**What it tests:**
- ✅ Kubernetes client initialization
- ✅ Namespace creation and management
- ✅ RBAC resource creation (ServiceAccount, Role, RoleBinding, ResourceQuota)
- ✅ ServiceAccount token generation
- ✅ ClusterRoleBinding subject mutations (capacity checker)
- ✅ Registration HTTP handler with real K8s resources
- ✅ Server lifecycle (create, list, start, stop, delete)

**Duration:** ~1-2 minutes (creates real resources and cleans up)

**Important:** These tests create and delete resources in your cluster. They automatically clean up on exit.

**Run:**
```bash
go test -p 1 -tags=integration ./internal/...
```

### 3. Master Test Suite (`test-all.sh`)

Runs Helm chart validation and Go integration tests in sequence and reports overall status.

## Understanding Test Results

### Success

```
========================================
   ALL TESTS PASSED! ✓
========================================
```

All manifests are valid and RBAC is properly configured.

### Failure

```
========================================
   SOME TESTS FAILED! ✗
========================================

Please review the failed tests above.
```

Scroll up to see which specific tests failed. Each failed test shows:
- Test name
- Expected vs actual result
- Error details (for manifest validation)

## CI/CD Integration

Tests run automatically in GitHub Actions on:
- Push to `main` branch
- Pull requests to `main` branch
- Changes to `charts/kubecraft-control-plane/**` or workflow files

**Workflow file:** `.github/workflows/test-manifests.yml` (Helm lint + template dry-run)

**View results:** GitHub Actions tab in your repository

## Common Issues & Troubleshooting

### Issue: "connection refused" errors

**Cause:** No Kubernetes cluster running or kubectl not configured

**Fix:**
```bash
# Check if cluster is running
kubectl cluster-info

# If not, start a test cluster
k3d cluster create kubecraft-test --agents 1
```

### Issue: "namespace already exists" errors

**Cause:** Previous test run didn't clean up properly

**Fix:**
```bash
# Manual cleanup
kubectl delete namespace mc-alice mc-bob --ignore-not-found
kubectl delete clusterrolebinding kc-users-capacity-check --ignore-not-found
```

### Issue: Tests pass locally but fail in CI

**Cause:** Kubernetes version differences or timing issues

**Fix:**
- Check GitHub Actions logs for specific error
- Ensure manifests use stable API versions (apps/v1, not apps/v1beta1)
- Add sleep/wait commands if timing-related

### Issue: "ClusterRoleBinding not found" in RBAC tests

**Cause:** Control-plane Helm chart not installed before running tests

**Fix:**
```bash
# Install control-plane chart first
helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane

# Then run tests
go test -p 1 -tags=integration ./internal/...
```

## Test Development

### Adding Helm Chart Tests

Add validation to CI or local scripts:

```bash
# Lint the chart
helm lint ./charts/kubecraft-control-plane

# Render and dry-run against cluster
helm template kubecraft-control-plane ./charts/kubecraft-control-plane | kubectl apply --dry-run=client -f -

# Verify specific templates
helm template kubecraft-control-plane ./charts/kubecraft-control-plane | grep -A 5 "kind: ClusterRole"
```

### Adding Go Integration Tests

Write tests in the relevant `internal/` package with the `integration` build tag:

```go
//go:build integration

package k8s

func TestMyFeature(t *testing.T) {
    client := GetTestClient(t)
    username := UniqueUsername()
    defer CleanupNamespace(t, client, username)

    // Your test code here
}
```

## Best Practices

1. **Always validate Helm charts before committing**
   ```bash
   helm lint ./charts/kubecraft-control-plane
   ```

2. **Run full test suite before pushing**
   ```bash
   ./scripts/test-all.sh
   ```

3. **Use a dedicated test cluster** (don't test on production!)

4. **Clean up manually if tests fail mid-execution**
   ```bash
   kubectl delete namespace mc-alice mc-bob --ignore-not-found
   kubectl delete clusterrolebinding kc-users-capacity-check --ignore-not-found
   ```

5. **Check CI results** before merging pull requests

## Test Coverage

Current test coverage:

| Component | Coverage | Notes |
|-----------|----------|-------|
| Helm chart validation | 100% | Lint + dry-run |
| RBAC permissions | ~95% | Core permissions tested |
| Namespace isolation | 100% | Cross-user access blocked |
| Resource creation | 80% | Basic server creation tested |

## Future Improvements

- [ ] Add yamllint for style consistency
- [ ] Add kubeconform for strict schema validation
- [ ] Add performance tests (server startup time)
- [ ] Add load tests (multiple concurrent users)
- [x] Add integration tests for registration service (Phase 2.5) ✅
- [ ] Add CLI tests (Phase 3+)

## Questions?

If you encounter issues not covered here:
1. Check test output carefully (error messages are usually helpful)
2. Verify your cluster is running: `kubectl get nodes`
3. Check manifest syntax manually: `kubectl apply --dry-run=client -f <file>`
4. Review GitHub Actions logs if CI is failing

---

**Last updated:** 2025-01-04
**Maintained by:** Kubecraft Team

---

# Go Testing Guide

This section covers the Go tests for the `internal/` packages (Phase 2.5 code).

## Go Test Structure

```
internal/
├── config/
│   └── constants_test.go      # Unit tests for constants
├── k8s/
│   ├── client_test.go          # Integration tests (//go:build integration)
│   ├── namespace_test.go       # Integration tests (//go:build integration)
│   ├── rbac_test.go            # Integration tests (//go:build integration)
│   ├── token_test.go           # Integration tests (//go:build integration)
│   └── helpers_test.go         # Shared test utilities (//go:build integration)
└── registration/
    ├── validator_test.go       # Unit tests (no build tag)
    ├── handler_test.go         # Integration tests (//go:build integration)
    └── helpers_test.go         # Test helpers (//go:build integration)
```

## Build Tags

Tests are separated using Go build tags:

| Tag | Description | Cluster Required |
|-----|-------------|------------------|
| (none) | Unit tests - run by default | No |
| `integration` | Integration tests - require `-tags=integration` | Yes |

Files with `//go:build integration` at the top are excluded from normal `go test` runs.

## Quick Start

### Run Unit Tests Only (No Cluster Needed)

```bash
go test -v ./internal/...
```

This runs only tests WITHOUT the `integration` build tag:
- `internal/config/constants_test.go`
- `internal/registration/validator_test.go`

### Run All Tests Including Integration (Requires Cluster)

```bash
go test -v -p 1 -tags=integration ./internal/...
```

This runs ALL tests (unit + integration).

## Prerequisites

### For Unit Tests

No prerequisites - these run anywhere with Go installed.

### For Integration Tests

1. **Kubernetes cluster running**
   ```bash
   k3d cluster create go-test-cluster --agents 1
   ```

2. **Control-plane Helm chart installed**
   ```bash
   helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane
   ```

3. **KUBECONFIG set** (or use default `~/.kube/config`)

## Test Categories

### **Unit Tests** (no build tag)

**Packages:** `internal/config/`, `internal/registration/` (validator only)

**What they test:**
- Constant values (MaxUsers, ports, etc.)
- Resource names
- Token expiry calculation
- NodePort range validation
- Username validation logic

**Run:**
```bash
go test -v ./internal/...
```

**Duration:** < 1 second

---

### **Integration Tests** (`//go:build integration`)

**Packages:** `internal/k8s/`, `internal/registration/` (handler only)

**What they test:**
- Client initialization (`client_test.go`)
- Namespace creation and management (`namespace_test.go`)
- RBAC resource creation (`rbac_test.go`)
- ServiceAccount token generation (`token_test.go`)
- HTTP handler with real K8s resources (`handler_test.go`)

**Run:**
```bash
go test -v -p 1 -tags=integration ./internal/...
```

**Duration:** ~30-60 seconds (creates real resources)

**Important:** Tests automatically clean up resources after completion.

---

## Test File Descriptions

### **constants_test.go** (Unit Tests)

Tests all configuration constants:

- `TestConstants_UserLimits` - MaxUsers, username length constraints
- `TestConstants_NetworkConfig` - Ports and NodePort range
- `TestConstants_NodePortRange` - Validates range can fit all users
- `TestConstants_ResourceNames` - Namespace prefixes, role names
- `TestConstants_CommonLabel` - Label key/value/selector consistency
- `TestConstants_ReservedNames` - Reserved username list
- `TestConstants_TokenExpiry` - 5-year expiration calculation
- `TestConstants_ResourceLimits` - CPU/memory limits

**Example:**
```bash
go test -v ./internal/config -run TestConstants_TokenExpiry
```

---

### **client_test.go** (Integration Tests)

Tests Kubernetes client initialization:

- `TestNewClientFromKubeConfig_Success` - Loads valid kubeconfig
- `TestNewClientFromKubeConfig_InvalidPath` - Handles missing config
- `TestNewInClusterClient_OutsideCluster` - Fails gracefully outside cluster
- `TestClient_GetClientset` - Clientset accessor works

**Example:**
```bash
go test -v -p 1 -tags=integration ./internal/k8s -run TestClient
```

---

### **namespace_test.go** (Integration Tests)

Tests namespace operations:

- `TestCreateNamespace_Success` - Creates namespace with correct labels
- `TestCreateNamespace_AlreadyExists` - Detects duplicates
- `TestCreateNamespace_LabelsCorrect` - Verifies `app=kubecraft` labels
- `TestNamespaceExists_ReturnsTrue/False` - Existence checks
- `TestCountUserNamespaces_ReturnsCorrectCount` - Counts only kubecraft namespaces

**Example:**
```bash
go test -v -p 1 -tags=integration ./internal/k8s -run TestNamespace
```

---

### **rbac_test.go** (Integration Tests)

Tests RBAC resource creation:

- `TestCreateServiceAccount_Success` - Creates SA with labels
- `TestCreateRole_Success` - Creates Role with correct permissions
- `TestCreateRoleBinding_Success` - Binds SA to Role
- `TestCreateResourceQuota_Success` - Applies compute limits
- `TestAddUserToCapacityChecker_Success` - Adds user to ClusterRoleBinding
- `TestAddUserToCapacityChecker_Duplicate` - Prevents duplicates

**Example:**
```bash
go test -v -p 1 -tags=integration ./internal/k8s -run TestRBAC
```

---

### **token_test.go** (Integration Tests)

Tests ServiceAccount token generation:

- `TestGenerateToken_Success` - Generates valid token
- `TestGenerateToken_ValidJWT` - Token is valid JWT format
- `TestGenerateToken_NonexistentServiceAccount` - Fails for missing SA
- `TestGenerateToken_ExpirationCorrect` - Token expires in ~5 years

**Example:**
```bash
go test -v -p 1 -tags=integration ./internal/k8s -run TestGenerateToken
```

---

### **validator_test.go** (Unit Tests)

Tests username validation logic:

- `TestValidateUsername_Success` - Valid usernames (6 cases)
- `TestValidateUsername_TooShort` - Rejects short usernames (3 cases)
- `TestValidateUsername_TooLong` - Rejects long usernames
- `TestValidateUsername_InvalidCharacters` - Rejects special chars (9 cases)
- `TestValidateUsername_MustStartWithLetter` - Rejects number-prefix (4 cases)
- `TestValidateUsername_ReservedNames` - Rejects reserved names (5 cases)
- `TestValidateUsername_EdgeCases` - Boundary tests (5 cases)

**Example:**
```bash
go test -v ./internal/registration -run TestValidateUsername
```

**Duration:** < 1 second

---

### **handler_test.go** (Integration Tests)

Tests registration HTTP handler:

- `TestHandler_MethodNotAllowed` - Rejects non-POST requests
- `TestHandler_InvalidJSON` - Handles malformed JSON
- `TestHandler_InvalidUsername` - Integrates with validator
- `TestHandler_SuccessfulRegistration` - Creates all resources
- `TestHandler_DuplicateUsername` - Prevents duplicates
- `TestHandler_ResponseFormat` - Verifies JSON responses
- `TestHandler_CreatesAllResources` - Verifies namespace, RBAC, quota

**Example:**
```bash
go test -v -p 1 -tags=integration ./internal/registration -run TestHandler
```

**Duration:** ~30-60 seconds (requires cluster + RBAC)

---

## Running with Coverage

```bash
# Unit tests coverage
go test -v -coverprofile=coverage.out ./internal/config/... ./internal/registration/...

# All tests coverage (requires cluster)
go test -v -p 1 -tags=integration -coverprofile=coverage.out \
  ./internal/config/... \
  ./internal/k8s/... \
  ./internal/registration/...

# View coverage in terminal
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

**Note:** Coverage must be collected on specific packages that have test files. Using `./internal/...` will fail with `covdata` errors for packages without tests (like `internal/cli`).

## Running with Race Detection

```bash
# Unit tests with race detection
go test -v -race ./internal/...

# All tests with race detection (requires cluster)
go test -v -race -p 1 -tags=integration ./internal/...
```

## CI/CD Integration

Tests run automatically in GitHub Actions via two separate workflows:

### Unit Tests Workflow

**Workflow:** `.github/workflows/test-unit.yml`

**Triggers:**
- Push to `main` branch
- Pull requests to `main`
- Changes to `internal/**`, `cmd/**`, `go.mod`, `go.sum`

**What it runs:**
```bash
go test -v -race -coverprofile=coverage.out \
  ./internal/config/... \
  ./internal/registration/...
```

**Duration:** ~30 seconds (no cluster needed)

---

### Integration Tests Workflow

**Workflow:** `.github/workflows/test-integration.yml`

**Triggers:**
- Push to `main` branch
- Pull requests to `main`
- Changes to `internal/**`, `cmd/**`, `go.mod`, `go.sum`

**What it runs:**
```bash
go test -v -race -p 1 -tags=integration -coverprofile=coverage.out \
  ./internal/config/... \
  ./internal/k8s/... \
  ./internal/registration/...
```

**Duration:** ~2-3 minutes (spins up k3d cluster)

## Common Issues & Troubleshooting

### Issue: "no kubeconfig found"

**Cause:** Integration tests can't find kubeconfig

**Fix:**
```bash
# Set KUBECONFIG explicitly
export KUBECONFIG=~/.kube/config

# Or create test cluster
k3d cluster create go-test-cluster
```

---

### Issue: "ClusterRole not found"

**Cause:** Control-plane Helm chart not installed

**Fix:**
```bash
helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane
```

---

### Issue: Tests fail with "connection refused"

**Cause:** No Kubernetes cluster running

**Fix:**
```bash
# Verify cluster is running
kubectl cluster-info

# If not, start cluster
k3d cluster create go-test-cluster
```

---

### Issue: "namespace already exists" errors

**Cause:** Previous test run didn't clean up properly

**Fix:**
```bash
# List test namespaces
kubectl get ns -l app=kubecraft

# Delete manually
kubectl delete ns -l app=kubecraft

# Or delete specific namespace
kubectl delete ns mc-testuser-123456
```

---

## Test Helpers

Test helper functions are defined in `*_test.go` files within each package:

- `internal/k8s/helpers_test.go` - K8s test utilities
- `internal/registration/helpers_test.go` - Registration test utilities

| Function | Package | Purpose |
|----------|---------|---------|
| `GetTestClient(t)` | k8s | Initialize k8s client from kubeconfig |
| `UniqueUsername()` | k8s | Generate unique test username |
| `CleanupNamespace(t, client, username)` | k8s | Delete namespace and wait |
| `CleanupClusterRoleBinding(t, client, username)` | k8s | Remove user from CRB |
| `RequireSystemRBAC(t, client)` | k8s | Assert ClusterRole/Binding exist (Helm-installed) |
| `WaitForServiceAccount(t, client, ns, name)` | k8s | Wait for SA to be ready |

**Example usage:**
```go
//go:build integration

package k8s

func TestMyFeature(t *testing.T) {
    client := GetTestClient(t)
    username := UniqueUsername()
    defer CleanupNamespace(t, client, username)

    // Your test code here
}
```

---

## Best Practices

1. **Always clean up** - Use `defer testutil.CleanupNamespace()` in every test
2. **Use unique names** - Call `testutil.UniqueUsername()` to avoid conflicts
3. **Wait for resources** - Use `WaitForServiceAccount()` before generating tokens
4. **Test in isolation** - Each test creates its own namespace
5. **Check errors** - Verify error messages, not just error existence

---

## Test Coverage Goals

| Package | Current Coverage | Target | Status |
|---------|-----------------|--------|--------|
| `internal/config` | ~95% | 90% | ✅ Excellent |
| `internal/k8s` | ~70% | 70% | ✅ Good |
| `internal/registration` | ~85% | 80% | ✅ Excellent |
| **Overall** | ~80% | 70% | ✅ Meeting goals |

---

## Next Steps

- [ ] Add tests for Phase 3 (CLI tool)
- [x] Add tests for Phase 2.5 (registration service) ✅
- [ ] Add performance benchmarks
- [ ] Add load tests (multiple concurrent users)

---

**Last updated:** 2026-01-25
**Test files:** 8 files, ~1400 lines of test code
**Total tests:** ~50 test functions
**Build tags:** Unit tests (no tag), Integration tests (`//go:build integration`)

