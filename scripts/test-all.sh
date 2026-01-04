#!/bin/bash

# Master Test Script for Kubecraft
# Runs all validation and functional tests

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

print_header() {
    echo -e "\n${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}\n"
}

# Track overall status
OVERALL_PASSED=true

print_header "Kubecraft Complete Test Suite"

echo "Running all tests..."
echo ""

# ========================================
# Phase 1: Manifest Validation
# ========================================

print_header "Phase 1: Manifest Validation"

if "${SCRIPT_DIR}/test-manifests.sh"; then
    echo -e "\n${GREEN}✓ Manifest tests passed${NC}"
else
    echo -e "\n${RED}✗ Manifest tests failed${NC}"
    OVERALL_PASSED=false
fi

# ========================================
# Phase 2: RBAC Functional Tests
# ========================================

print_header "Phase 2: RBAC Functional Tests"

if "${SCRIPT_DIR}/test-rbac.sh"; then
    echo -e "\n${GREEN}✓ RBAC tests passed${NC}"
else
    echo -e "\n${RED}✗ RBAC tests failed${NC}"
    OVERALL_PASSED=false
fi

# ========================================
# Final Summary
# ========================================

print_header "Final Test Summary"

if [ "$OVERALL_PASSED" = true ]; then
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}   ALL TEST SUITES PASSED! ✓${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "✓ Manifest validation passed"
    echo "✓ RBAC functional tests passed"
    echo ""
    echo "Your Kubecraft setup is ready!"
    exit 0
else
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}   SOME TEST SUITES FAILED! ✗${NC}"
    echo -e "${RED}========================================${NC}"
    echo ""
    echo "Please review the test output above."
    exit 1
fi
