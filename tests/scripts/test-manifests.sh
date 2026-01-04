#!/bin/bash

# Manifest Testing Script for Kubecraft
# Validates YAML syntax, structure, and template substitution

set -euo pipefail

# Get script directory for relative paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_DIR="${SCRIPT_DIR}/../../manifests"

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

# Validate YAML syntax and K8s API compliance
validate_manifest() {
    local file=$1
    local description=$2

    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    echo -n "  [TEST] $description ... "

    if kubectl apply --dry-run=client -f "$file" &>/dev/null; then
        echo -e "${GREEN}✓ PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ FAIL${NC}"
        kubectl apply --dry-run=client -f "$file" 2>&1 | head -5 | sed 's/^/    /'
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Validate manifest with template substitution
validate_templated_manifest() {
    local file=$1
    local description=$2
    local username=${3:-"testuser"}
    local servername=${4:-"testserver"}

    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    echo -n "  [TEST] $description ... "

    # Create temporary file with substitutions
    local temp_file=$(mktemp)
    sed -e "s/{username}/$username/g" -e "s/{servername}/$servername/g" "$file" > "$temp_file"

    if kubectl apply --dry-run=client -f "$temp_file" &>/dev/null; then
        echo -e "${GREEN}✓ PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        rm -f "$temp_file"
    else
        echo -e "${RED}✗ FAIL${NC}"
        echo "    Substituted manifest:"
        kubectl apply --dry-run=client -f "$temp_file" 2>&1 | head -5 | sed 's/^/    /'
        rm -f "$temp_file"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Check if manifest contains required field
check_field_exists() {
    local file=$1
    local field=$2
    local description=$3

    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    echo -n "  [TEST] $description ... "

    if grep -q -- "$field" "$file"; then
        echo -e "${GREEN}✓ PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ FAIL${NC}"
        echo -e "    Field '$field' not found in $file"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Main test execution
main() {
    print_header "Kubecraft Manifest Validation Suite"

    echo "This script will validate:"
    echo "  • YAML syntax"
    echo "  • Kubernetes API compliance"
    echo "  • Template placeholder substitution"
    echo "  • Required fields and labels"
    echo "  • Resource references"
    echo ""

    # ========================================
    # SYSTEM TEMPLATES
    # ========================================

    print_header "Phase 1: System Templates"

    print_section "Testing Registration Service Manifests"

    validate_manifest \
        "${MANIFEST_DIR}/registration-templates/registration-namespace.yaml" \
        "registration-namespace.yaml is valid"

    validate_manifest \
        "${MANIFEST_DIR}/registration-templates/registration-serviceaccount.yaml" \
        "registration-serviceaccount.yaml is valid"

    validate_manifest \
        "${MANIFEST_DIR}/registration-templates/registration-clusterrole.yaml" \
        "registration-clusterrole.yaml is valid"

    validate_manifest \
        "${MANIFEST_DIR}/registration-templates/registration-clusterrolebinding.yaml" \
        "registration-clusterrolebinding.yaml is valid"

    validate_manifest \
        "${MANIFEST_DIR}/registration-templates/registration-deployment.yaml" \
        "registration-deployment.yaml is valid"

    validate_manifest \
        "${MANIFEST_DIR}/registration-templates/registration-service.yaml" \
        "registration-service.yaml is valid"

    print_section "Testing System RBAC Manifests"

    validate_manifest \
        "${MANIFEST_DIR}/system-templates/clusterrole.yaml" \
        "clusterrole.yaml is valid"

    validate_manifest \
        "${MANIFEST_DIR}/system-templates/clusterrolebinding.yaml" \
        "clusterrolebinding.yaml is valid"

    # ========================================
    # USER TEMPLATES (with substitution)
    # ========================================

    print_header "Phase 2: User Templates"

    print_section "Testing User Namespace Manifests"

    validate_templated_manifest \
        "${MANIFEST_DIR}/user-templates/namespace.yaml" \
        "namespace.yaml is valid (with substitution)" \
        "alice"

    validate_templated_manifest \
        "${MANIFEST_DIR}/user-templates/serviceaccount.yaml" \
        "serviceaccount.yaml is valid (with substitution)" \
        "alice"

    validate_templated_manifest \
        "${MANIFEST_DIR}/user-templates/role.yaml" \
        "role.yaml is valid (with substitution)" \
        "alice"

    validate_templated_manifest \
        "${MANIFEST_DIR}/user-templates/rolebinding.yaml" \
        "rolebinding.yaml is valid (with substitution)" \
        "alice"

    validate_templated_manifest \
        "${MANIFEST_DIR}/user-templates/resourcequota.yaml" \
        "resourcequota.yaml is valid (with substitution)" \
        "alice"

    # ========================================
    # SERVER TEMPLATES (with substitution)
    # ========================================

    print_header "Phase 3: Server Templates"

    print_section "Testing Minecraft Server Manifests"

    validate_templated_manifest \
        "${MANIFEST_DIR}/server-templates/statefulset.yaml" \
        "statefulset.yaml is valid (with substitution)" \
        "alice" \
        "myserver"

    validate_templated_manifest \
        "${MANIFEST_DIR}/server-templates/service.yaml" \
        "service.yaml is valid (with substitution)" \
        "alice" \
        "myserver"

    # ========================================
    # REQUIRED FIELDS CHECKS
    # ========================================

    print_header "Phase 4: Required Fields & Labels"

    print_section "Checking User Template Labels"

    check_field_exists \
        "${MANIFEST_DIR}/user-templates/namespace.yaml" \
        "app: kubecraft" \
        "namespace.yaml has 'app: kubecraft' label"

    check_field_exists \
        "${MANIFEST_DIR}/user-templates/serviceaccount.yaml" \
        "app: kubecraft" \
        "serviceaccount.yaml has 'app: kubecraft' label"

    check_field_exists \
        "${MANIFEST_DIR}/user-templates/role.yaml" \
        "minecraft-manager" \
        "role.yaml defines 'minecraft-manager' role"

    print_section "Checking Server Template Configuration"

    check_field_exists \
        "${MANIFEST_DIR}/server-templates/statefulset.yaml" \
        "image: hasanbaig786/kubecraft" \
        "statefulset.yaml uses correct image"

    check_field_exists \
        "${MANIFEST_DIR}/server-templates/statefulset.yaml" \
        "memory: \"768Mi\"" \
        "statefulset.yaml has memory request (768Mi)"

    check_field_exists \
        "${MANIFEST_DIR}/server-templates/statefulset.yaml" \
        "memory: \"1Gi\"" \
        "statefulset.yaml has memory limit (1Gi)"

    check_field_exists \
        "${MANIFEST_DIR}/server-templates/service.yaml" \
        "type: NodePort" \
        "service.yaml is NodePort type"

    check_field_exists \
        "${MANIFEST_DIR}/server-templates/service.yaml" \
        "port: 25565" \
        "service.yaml exposes port 25565"

    # ========================================
    # RBAC REFERENCE CHECKS
    # ========================================

    print_header "Phase 5: Resource References"

    print_section "Checking RBAC References"

    check_field_exists \
        "${MANIFEST_DIR}/user-templates/rolebinding.yaml" \
        "kind: Role" \
        "rolebinding.yaml references a Role"

    check_field_exists \
        "${MANIFEST_DIR}/user-templates/rolebinding.yaml" \
        "name: minecraft-manager" \
        "rolebinding.yaml references 'minecraft-manager' role"

    check_field_exists \
        "${MANIFEST_DIR}/user-templates/role.yaml" \
        "statefulsets" \
        "role.yaml grants statefulsets permissions"

    check_field_exists \
        "${MANIFEST_DIR}/user-templates/role.yaml" \
        "persistentvolumeclaims" \
        "role.yaml grants PVC permissions"

    check_field_exists \
        "${MANIFEST_DIR}/registration-templates/registration-clusterrolebinding.yaml" \
        "kc-registration-admin" \
        "registration ClusterRoleBinding references correct ClusterRole"

    check_field_exists \
        "${MANIFEST_DIR}/system-templates/clusterrole.yaml" \
        "kc-capacity-checker" \
        "capacity checker ClusterRole has correct name"

    # ========================================
    # REGISTRATION SERVICE CHECKS
    # ========================================

    print_header "Phase 6: Registration Service Configuration"

    print_section "Checking Registration Service Settings"

    check_field_exists \
        "${MANIFEST_DIR}/registration-templates/registration-service.yaml" \
        "nodePort: 30099" \
        "registration service uses NodePort 30099"

    check_field_exists \
        "${MANIFEST_DIR}/registration-templates/registration-deployment.yaml" \
        "MAX_USERS" \
        "registration deployment defines MAX_USERS env var"

    check_field_exists \
        "${MANIFEST_DIR}/registration-templates/registration-clusterrole.yaml" \
        '"namespaces"' \
        "registration ClusterRole can manage namespaces"

    check_field_exists \
        "${MANIFEST_DIR}/registration-templates/registration-clusterrole.yaml" \
        '"serviceaccounts/token"' \
        "registration ClusterRole can create tokens"

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
        echo "All manifests are valid:"
        echo "  ✓ YAML syntax is correct"
        echo "  ✓ Kubernetes API compliance verified"
        echo "  ✓ Template substitution works"
        echo "  ✓ Required fields present"
        echo "  ✓ Resource references are correct"
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
