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
./tests/scripts/test-all.sh
```

This will run:
1. Manifest validation tests
2. RBAC functional tests

### Run Individual Test Suites

**Manifest validation only:**
```bash
./tests/scripts/test-manifests.sh
```

**RBAC functional tests only:**
```bash
./tests/scripts/test-rbac.sh
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

Edit `tests/scripts/test-manifests.sh` and add tests using these functions:

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

Edit `tests/scripts/test-rbac.sh` and add tests using:

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
   ./tests/scripts/test-manifests.sh
   ```

2. **Run full test suite before pushing**
   ```bash
   ./tests/scripts/test-all.sh
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
