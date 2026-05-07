#!/bin/bash

# Master Test Script for Kubecraft
# Runs Helm validation and Go integration tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

GREEN='\033[0;32m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

print_header() {
    echo -e "\n${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}\n"
}

OVERALL_PASSED=true

print_header "Kubecraft Complete Test Suite"

# ========================================
# Phase 1: Helm Validation
# ========================================

print_header "Phase 1: Helm Validation"

echo "Running helm lint..."
if helm lint ./charts/kubecraft-control-plane; then
    echo -e "\n${GREEN}✓ Helm lint passed${NC}"
else
    echo -e "\n${RED}✗ Helm lint failed${NC}"
    OVERALL_PASSED=false
fi

echo "Running helm template dry-run..."
if helm template kubecraft-control-plane ./charts/kubecraft-control-plane | kubectl apply --dry-run=client -f -; then
    echo -e "\n${GREEN}✓ Helm template dry-run passed${NC}"
else
    echo -e "\n${RED}✗ Helm template dry-run failed${NC}"
    OVERALL_PASSED=false
fi

# ========================================
# Phase 2: Install control-plane chart
# ========================================

print_header "Phase 2: Install Control-Plane Chart"

echo "Installing control-plane Helm chart..."
if helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane; then
    echo -e "\n${GREEN}✓ Control-plane chart installed${NC}"
else
    echo -e "\n${RED}✗ Control-plane chart installation failed${NC}"
    OVERALL_PASSED=false
fi

# ========================================
# Phase 3: Go Integration Tests
# ========================================

print_header "Phase 3: Go Integration Tests"

echo "Running integration tests..."
if go test -v -race -p 1 -tags=integration ./internal/...; then
    echo -e "\n${GREEN}✓ Integration tests passed${NC}"
else
    echo -e "\n${RED}✗ Integration tests failed${NC}"
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
    echo "✓ Helm validation passed"
    echo "✓ Integration tests passed"
    exit 0
else
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}   SOME TEST SUITES FAILED! ✗${NC}"
    echo -e "${RED}========================================${NC}"
    echo ""
    echo "Please review the test output above."
    exit 1
fi
