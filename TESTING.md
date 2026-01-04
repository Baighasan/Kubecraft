# Kubecraft Testing Guide

This document explains how to run tests for the Kubecraft project, including manifest validation and RBAC functional tests.

## Test Structure

```
tests/
└── scripts/
    ├── test-manifests.sh    # Validates YAML syntax and Kubernetes compliance
    ├── test-rbac.sh         # Tests RBAC permissions and namespace isolation
    └── test-all.sh          # Runs all tests in sequence

scripts/                     # Helper scripts (used by tests)
├── create-user.sh           # Helper: Creates test users
├── delete-user.sh           # Helper: Deletes test users
└── apply-system-rbac.sh     # Helper: Applies cluster-wide RBAC
```

## Quick Start

### Run All Tests (Recommended)

```bash
./scripts/test-all.sh
```

This will run:
1. Manifest validation tests
2. RBAC functional tests

### Run Individual Test Suites

**Manifest validation only:**
```bash
./scripts/test-manifests.sh
```

**RBAC functional tests only:**
```bash
./scripts/test-rbac.sh
```

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

### 1. Manifest Validation (`test-manifests.sh`)

**What it tests:**
- ✅ YAML syntax validation
- ✅ Kubernetes API compliance (using `kubectl apply --dry-run`)
- ✅ Template placeholder substitution (`{username}`, `{servername}`)
- ✅ Required fields presence (labels, resource limits, ports)
- ✅ Resource references (RoleBinding → Role, etc.)

**Test phases:**
1. **System Templates** - Registration service and cluster RBAC
2. **User Templates** - Namespace, ServiceAccount, Role, RoleBinding, ResourceQuota
3. **Server Templates** - StatefulSet and Service manifests
4. **Required Fields** - Labels, image names, resource limits
5. **Resource References** - RBAC references, naming consistency
6. **Registration Config** - NodePort settings, environment variables

**Duration:** ~30 seconds

**Output example:**
```
========================================
Phase 1: System Templates
========================================

>>> Testing Registration Service Manifests
  [TEST] registration-namespace.yaml is valid ... ✓ PASS
  [TEST] registration-serviceaccount.yaml is valid ... ✓ PASS
  ...

========================================
Test Results Summary
========================================
Total Tests: 45
Passed:      45
Failed:      0

   ALL TESTS PASSED! ✓
```

### 2. RBAC Functional Tests (`test-rbac.sh`)

**What it tests:**
- ✅ Cluster-scoped permissions (capacity checking)
- ✅ Namespace-scoped permissions (server management)
- ✅ Namespace isolation (users can't access each other)
- ✅ Actual server creation (StatefulSet, Service)
- ✅ ResourceQuota enforcement
- ✅ Security boundaries

**Test phases:**
1. **Setup** - Apply system RBAC, create test users (alice, bob)
2. **Cluster-Scoped Permissions** - Verify users can list namespaces/services/pods
3. **Namespace-Scoped Permissions** - Verify users can manage their own resources
4. **Namespace Isolation** - Verify users cannot access each other's namespaces
5. **Functional Tests** - Actually create a Minecraft server
6. **ResourceQuota** - Verify compute limits are enforced
7. **Security Boundaries** - Verify users cannot escalate privileges

**Duration:** ~1-2 minutes (includes resource creation and cleanup)

**Important:** This test creates and deletes resources in your cluster. It automatically cleans up on exit.

### 3. Master Test Suite (`test-all.sh`)

Runs both test suites in sequence and reports overall status.

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
- Changes to `manifests/**` or test scripts

**Workflow file:** `.github/workflows/test-manifests.yml`

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

**Cause:** System RBAC not applied before running tests

**Fix:**
```bash
# Apply system RBAC first
./scripts/apply-system-rbac.sh

# Then run tests
./scripts/test-rbac.sh
```

## Test Development

### Adding New Manifest Tests

Edit `scripts/test-manifests.sh` and add tests using these functions:

```bash
# Validate static manifest
validate_manifest \
    "${MANIFEST_DIR}/path/to/manifest.yaml" \
    "Description of what you're testing"

# Validate templated manifest
validate_templated_manifest \
    "${MANIFEST_DIR}/path/to/template.yaml" \
    "Description" \
    "username" \
    "servername"

# Check if field exists
check_field_exists \
    "${MANIFEST_DIR}/path/to/manifest.yaml" \
    "field: value" \
    "Description"
```

### Adding New RBAC Tests

Edit `scripts/test-rbac.sh` and add tests using:

```bash
# Test if user CAN do something
test_result "User can do X" \
    "pass" \
    "kubectl auth can-i <verb> <resource> --as=system:serviceaccount:mc-$USER1:$USER1"

# Test if user CANNOT do something
test_result "User CANNOT do X" \
    "fail" \
    "kubectl auth can-i <verb> <resource> --as=system:serviceaccount:mc-$USER1:$USER1"
```

## Best Practices

1. **Always run tests before committing manifest changes**
   ```bash
   ./scripts/test-manifests.sh
   ```

2. **Run full test suite before pushing**
   ```bash
   ./scripts/test-all.sh
   ```

3. **Use a dedicated test cluster** (don't test on production!)

4. **Clean up manually if tests fail mid-execution**
   ```bash
   ./scripts/delete-user.sh alice
   ./scripts/delete-user.sh bob
   ```

5. **Check CI results** before merging pull requests

## Test Coverage

Current test coverage:

| Component | Coverage | Notes |
|-----------|----------|-------|
| System RBAC manifests | 100% | All files validated |
| User templates | 100% | All files validated |
| Server templates | 100% | All files validated |
| RBAC permissions | ~95% | Core permissions tested |
| Namespace isolation | 100% | Cross-user access blocked |
| Resource creation | 80% | Basic server creation tested |

## Future Improvements

- [ ] Add yamllint for style consistency
- [ ] Add kubeconform for strict schema validation
- [ ] Add performance tests (server startup time)
- [ ] Add load tests (multiple concurrent users)
- [ ] Add integration tests for registration service (Phase 2.5+)
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

This section covers the Go tests for the `pkg/` packages (Phase 2.5 code).

## Go Test Structure

```
tests/
└── pkg/                      # Go tests (mirrors pkg/ structure)
    ├── config/
    │   └── constants_test.go      # Unit tests for constants
    └── k8s/
        ├── client_test.go          # Client initialization tests
        ├── namespace_test.go       # Namespace operation tests
        ├── rbac_test.go            # RBAC creation tests
        ├── token_test.go           # Token generation tests
        └── testutil/
            └── helpers.go          # Shared test utilities
```

## Quick Start

### Run All Go Tests

```bash
go test -v ./pkg/...
```

### Run Specific Package Tests

```bash
# Unit tests (no cluster needed)
go test -v ./pkg/config

# Integration tests (requires k8s cluster)
go test -v ./pkg/k8s
```

## Prerequisites

### For Unit Tests (constants)

No prerequisites - these run anywhere.

### For Integration Tests (k8s package)

1. **Kubernetes cluster running**
   ```bash
   k3d cluster create go-test-cluster --agents 1
   ```

2. **System RBAC applied**
   ```bash
   kubectl apply -f manifests/system-templates/clusterrole.yaml
   kubectl apply -f manifests/system-templates/clusterrolebinding.yaml
   ```

3. **KUBECONFIG set** (or use default `~/.kube/config`)

## Test Categories

### **Unit Tests** (`pkg/config/`)

**What they test:**
- Constant values (MaxUsers, ports, etc.)
- Resource names
- Token expiry calculation
- NodePort range validation

**Run:**
```bash
go test -v ./pkg/config
```

**Duration:** < 1 second

---

### **Integration Tests** (`pkg/k8s/`)

**What they test:**
- Client initialization (`client_test.go`)
- Namespace creation and management (`namespace_test.go`)
- RBAC resource creation (`rbac_test.go`)
- ServiceAccount token generation (`token_test.go`)

**Run:**
```bash
go test -v ./pkg/k8s
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
go test -v ./pkg/config -run TestConstants_TokenExpiry
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
go test -v ./pkg/k8s -run TestClient
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
go test -v ./pkg/k8s -run TestNamespace
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
go test -v ./pkg/k8s -run TestRBAC
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
go test -v ./pkg/k8s -run TestGenerateToken
```

---

## Running with Coverage

```bash
# Generate coverage report
go test -v -coverprofile=coverage.out ./pkg/...

# View coverage in terminal
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

## Running with Race Detection

```bash
# Detect race conditions
go test -v -race ./pkg/...
```

## CI/CD Integration

Tests run automatically in GitHub Actions:

**Workflow:** `.github/workflows/test-go.yml`

**Triggers:**
- Push to `main` branch
- Pull requests to `main`
- Changes to `pkg/**`, `pkg/**`, `go.mod`, `go.sum`

**Jobs:**
1. **unit-tests** - Runs constant tests (fast)
2. **integration-tests** - Runs k8s tests in k3d cluster
3. **test-summary** - Reports overall status

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

**Cause:** System RBAC not applied

**Fix:**
```bash
kubectl apply -f manifests/system-templates/clusterrole.yaml
kubectl apply -f manifests/system-templates/clusterrolebinding.yaml
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

## Test Helpers (`testutil/helpers.go`)

Shared utilities for integration tests:

| Function | Purpose |
|----------|---------|
| `GetTestClient(t)` | Initialize k8s client from kubeconfig |
| `UniqueUsername()` | Generate unique test username |
| `CleanupNamespace(t, client, username)` | Delete namespace and wait |
| `CleanupClusterRoleBinding(t, client, username)` | Remove user from CRB |
| `EnsureSystemRBAC(t, client)` | Create ClusterRole/Binding if missing |
| `WaitForServiceAccount(t, client, ns, name)` | Wait for SA to be ready |

**Example usage:**
```go
func TestMyFeature(t *testing.T) {
    client := testutil.GetTestClient(t)
    username := testutil.UniqueUsername()
    defer testutil.CleanupNamespace(t, client, username)
    
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
| `pkg/config` | ~95% | 90% | ✅ Excellent |
| `pkg/k8s` | ~70% | 70% | ✅ Good |
| **Overall** | ~75% | 70% | ✅ Meeting goals |

---

## Next Steps

- [ ] Add tests for Phase 3 (CLI tool)
- [ ] Add tests for Phase 2.5+ (registration service)
- [ ] Add performance benchmarks
- [ ] Add load tests (multiple concurrent users)

---

**Last updated:** 2026-01-04  
**Test files:** 5 files, ~800 lines of test code  
**Total tests:** ~30 test functions

