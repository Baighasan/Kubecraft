#!/bin/bash

# Comprehensive RBAC Testing Script for Kubecraft
# Tests user isolation, permissions, and security boundaries

set -euo pipefail

# Get script directory for relative paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELPER_SCRIPT_DIR="${SCRIPT_DIR}"
MANIFEST_DIR="${SCRIPT_DIR}/../manifests"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Test users
USER1="alice"
USER2="bob"

# Utility functions
print_header() {
    echo -e "\n${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}\n"
}

print_section() {
    echo -e "\n${BLUE}>>> $1${NC}"
}

test_result() {
    local test_name=$1
    local expected=$2  # "pass" or "fail"
    local command=$3

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    echo -n "  [TEST] $test_name ... "

    if eval "$command" &>/dev/null; then
        actual="pass"
    else
        actual="fail"
    fi

    if [ "$expected" = "$actual" ]; then
        echo -e "${GREEN}✓ PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ FAIL${NC}"
        echo -e "    Expected: $expected, Got: $actual"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Cleanup function
cleanup() {
    print_header "Cleaning Up Test Resources"

    echo "Deleting test users and namespaces..."
    "${HELPER_SCRIPT_DIR}/delete-user.sh" "$USER1" 2>/dev/null || echo "  $USER1 already deleted or doesn't exist"
    "${HELPER_SCRIPT_DIR}/delete-user.sh" "$USER2" 2>/dev/null || echo "  $USER2 already deleted or doesn't exist"

    echo -e "${GREEN}Cleanup complete${NC}"
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Main test execution
main() {
    print_header "Kubecraft RBAC Test Suite"

    echo "This script will test:"
    echo "  • System RBAC setup"
    echo "  • User namespace isolation"
    echo "  • Permission boundaries"
    echo "  • Actual server creation and management"
    echo ""

    # ========================================
    # SETUP PHASE
    # ========================================

    print_header "Phase 1: Setup"

    print_section "Applying System RBAC"
    if "${HELPER_SCRIPT_DIR}/apply-system-rbac.sh"; then
        echo -e "${GREEN}✓ System RBAC applied${NC}"
    else
        echo -e "${RED}✗ Failed to apply system RBAC${NC}"
        exit 1
    fi

    print_section "Creating Test Users"
    echo "Creating user: $USER1"
    if "${HELPER_SCRIPT_DIR}/create-user.sh" "$USER1"; then
        echo -e "${GREEN}✓ User $USER1 created${NC}"
    else
        echo -e "${RED}✗ Failed to create user $USER1${NC}"
        exit 1
    fi

    echo ""
    echo "Creating user: $USER2"
    if "${HELPER_SCRIPT_DIR}/create-user.sh" "$USER2"; then
        echo -e "${GREEN}✓ User $USER2 created${NC}"
    else
        echo -e "${RED}✗ Failed to create user $USER2${NC}"
        exit 1
    fi

    # Wait for service accounts to be ready
    echo ""
    echo "Waiting for ServiceAccounts to be ready..."
    sleep 2

    # ========================================
    # CLUSTER-SCOPED PERMISSIONS (ClusterRole)
    # ========================================

    print_header "Phase 2: Cluster-Scoped Permissions"

    print_section "Testing Capacity Check Permissions (ClusterRole: kc-capacity-checker)"

    # Positive tests - what users CAN do
    test_result "User can list namespaces" \
        "pass" \
        "kubectl auth can-i list namespaces --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can list services (cluster-wide)" \
        "pass" \
        "kubectl auth can-i list services --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can list pods (cluster-wide)" \
        "pass" \
        "kubectl auth can-i list pods --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can get namespaces" \
        "pass" \
        "kubectl auth can-i get namespaces --as=system:serviceaccount:mc-$USER1:$USER1"

    # Negative tests - what users CANNOT do
    test_result "User CANNOT create namespaces" \
        "fail" \
        "kubectl auth can-i create namespaces --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User CANNOT delete namespaces" \
        "fail" \
        "kubectl auth can-i delete namespaces --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User CANNOT create cluster roles" \
        "fail" \
        "kubectl auth can-i create clusterroles --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User CANNOT delete cluster role bindings" \
        "fail" \
        "kubectl auth can-i delete clusterrolebindings --as=system:serviceaccount:mc-$USER1:$USER1"

    # ========================================
    # NAMESPACE-SCOPED PERMISSIONS (Role)
    # ========================================

    print_header "Phase 3: Namespace-Scoped Permissions"

    print_section "Testing Permissions in Own Namespace (mc-$USER1)"

    # StatefulSet permissions
    test_result "User can create statefulsets in own namespace" \
        "pass" \
        "kubectl auth can-i create statefulsets -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can get statefulsets in own namespace" \
        "pass" \
        "kubectl auth can-i get statefulsets -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can list statefulsets in own namespace" \
        "pass" \
        "kubectl auth can-i list statefulsets -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can update statefulsets in own namespace" \
        "pass" \
        "kubectl auth can-i update statefulsets -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can patch statefulsets in own namespace" \
        "pass" \
        "kubectl auth can-i patch statefulsets -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    # Service permissions
    test_result "User can create services in own namespace" \
        "pass" \
        "kubectl auth can-i create services -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can delete services in own namespace" \
        "pass" \
        "kubectl auth can-i delete services -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can list services in own namespace" \
        "pass" \
        "kubectl auth can-i list services -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    # PVC permissions
    test_result "User can create PVCs in own namespace" \
        "pass" \
        "kubectl auth can-i create persistentvolumeclaims -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can delete PVCs in own namespace" \
        "pass" \
        "kubectl auth can-i delete persistentvolumeclaims -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    # Pod permissions (read-only)
    test_result "User can get pods in own namespace" \
        "pass" \
        "kubectl auth can-i get pods -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can list pods in own namespace" \
        "pass" \
        "kubectl auth can-i list pods -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User can get pod logs in own namespace" \
        "pass" \
        "kubectl auth can-i get pods/log -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User CANNOT delete pods in own namespace" \
        "fail" \
        "kubectl auth can-i delete pods -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User CANNOT create pods directly in own namespace" \
        "fail" \
        "kubectl auth can-i create pods -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    # ========================================
    # NAMESPACE ISOLATION
    # ========================================

    print_header "Phase 4: Namespace Isolation"

    print_section "Testing Cross-Namespace Access (User isolation)"

    test_result "User1 CANNOT create statefulsets in User2's namespace" \
        "fail" \
        "kubectl auth can-i create statefulsets -n mc-$USER2 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User1 CANNOT list statefulsets in User2's namespace" \
        "fail" \
        "kubectl auth can-i list statefulsets -n mc-$USER2 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User1 CANNOT delete services in User2's namespace" \
        "fail" \
        "kubectl auth can-i delete services -n mc-$USER2 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User1 CAN get pods in User2's namespace (capacity check)" \
        "pass" \
        "kubectl auth can-i get pods -n mc-$USER2 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User2 CANNOT create PVCs in User1's namespace" \
        "fail" \
        "kubectl auth can-i create persistentvolumeclaims -n mc-$USER1 --as=system:serviceaccount:mc-$USER2:$USER2"

    # ========================================
    # FUNCTIONAL TESTS
    # ========================================

    print_header "Phase 5: Functional Tests"

    print_section "Testing Actual Server Creation"

    # Create a test server for USER1
    SERVER_NAME="testserver"
    echo "Creating server '$SERVER_NAME' for user $USER1..."

    # Apply server manifests with substitutions
    sed -e "s/{username}/$USER1/g" -e "s/{servername}/$SERVER_NAME/g" \
        "${MANIFEST_DIR}/server-templates/statefulset.yaml" | \
        kubectl apply -f - &>/dev/null

    sed -e "s/{username}/$USER1/g" -e "s/{servername}/$SERVER_NAME/g" \
        "${MANIFEST_DIR}/server-templates/service.yaml" | \
        kubectl apply -f - &>/dev/null

    echo -e "${GREEN}✓ Server manifests applied${NC}"

    # Wait for StatefulSet to be created
    sleep 2

    # Verify resources exist
    echo ""
    echo "Verifying created resources..."

    if kubectl get statefulset "$SERVER_NAME" -n "mc-$USER1" &>/dev/null; then
        echo -e "${GREEN}✓ StatefulSet created${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ StatefulSet not found${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    if kubectl get service "$SERVER_NAME" -n "mc-$USER1" &>/dev/null; then
        echo -e "${GREEN}✓ Service created${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ Service not found${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    # Check service type
    SERVICE_TYPE=$(kubectl get service "$SERVER_NAME" -n "mc-$USER1" -o jsonpath='{.spec.type}')
    if [ "$SERVICE_TYPE" = "NodePort" ]; then
        echo -e "${GREEN}✓ Service is NodePort type${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ Service is not NodePort (found: $SERVICE_TYPE)${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    # Verify USER2 cannot access USER1's server
    echo ""
    print_section "Testing Server Isolation"

    test_result "User2 CANNOT delete User1's StatefulSet" \
        "fail" \
        "kubectl auth can-i delete statefulset $SERVER_NAME -n mc-$USER1 --as=system:serviceaccount:mc-$USER2:$USER2"

    test_result "User2 CANNOT update User1's Service" \
        "fail" \
        "kubectl auth can-i update service $SERVER_NAME -n mc-$USER1 --as=system:serviceaccount:mc-$USER2:$USER2"

    # ========================================
    # RESOURCE QUOTA TESTS
    # ========================================

    print_header "Phase 6: ResourceQuota Enforcement"

    print_section "Verifying ResourceQuota Exists"

    if kubectl get resourcequota -n "mc-$USER1" &>/dev/null; then
        echo -e "${GREEN}✓ ResourceQuota exists in mc-$USER1${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))

        # Show quota details
        echo ""
        echo "ResourceQuota details:"
        kubectl get resourcequota -n "mc-$USER1" -o yaml | grep -A 10 "hard:" || true
    else
        echo -e "${RED}✗ ResourceQuota not found${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    # ========================================
    # SECURITY TESTS
    # ========================================

    print_header "Phase 7: Security Boundaries"

    print_section "Testing System Resource Protection"

    test_result "User CAN access kube-system namespace (capacity check)" \
        "pass" \
        "kubectl auth can-i list pods -n kube-system --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User CANNOT delete own namespace" \
        "fail" \
        "kubectl auth can-i delete namespace mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User CANNOT create roles in own namespace" \
        "fail" \
        "kubectl auth can-i create roles -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User CANNOT create rolebindings in own namespace" \
        "fail" \
        "kubectl auth can-i create rolebindings -n mc-$USER1 --as=system:serviceaccount:mc-$USER1:$USER1"

    test_result "User CANNOT escalate privileges" \
        "fail" \
        "kubectl auth can-i '*' '*' --as=system:serviceaccount:mc-$USER1:$USER1"

    # ========================================
    # RESULTS SUMMARY
    # ========================================

    print_header "Test Results Summary"

    echo -e "Total Tests: ${CYAN}$TESTS_TOTAL${NC}"
    echo -e "Passed:      ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed:      ${RED}$TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}   ALL TESTS PASSED! ✓${NC}"
        echo -e "${GREEN}========================================${NC}"
        echo ""
        echo "RBAC is properly configured:"
        echo "  ✓ Users have correct permissions in their namespaces"
        echo "  ✓ Users are isolated from each other"
        echo "  ✓ Cluster-scoped permissions work for capacity checks"
        echo "  ✓ Security boundaries are enforced"
        exit 0
    else
        echo -e "${RED}========================================${NC}"
        echo -e "${RED}   SOME TESTS FAILED! ✗${NC}"
        echo -e "${RED}========================================${NC}"
        echo ""
        echo "Please review the failed tests above."
        exit 1
    fi
}

# Run main function
main
