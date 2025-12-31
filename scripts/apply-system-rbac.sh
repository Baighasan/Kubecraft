#!/bin/bash

# Exit on errors
set -euo pipefail

# Get script directory for relative paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYSTEM_MANIFEST_DIR="${SCRIPT_DIR}/../manifests/system-templates"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Applying System RBAC Manifests...${NC}"

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}ERROR: kubectl not found. Please install kubectl.${NC}"
    exit 1
fi

# Check if manifest directory exists
if [ ! -d "${SYSTEM_MANIFEST_DIR}" ]; then
    echo -e "${RED}ERROR: Manifest directory not found: ${SYSTEM_MANIFEST_DIR}${NC}"
    exit 1
fi

# Apply clusterrole.yaml
CLUSTERROLE="${SYSTEM_MANIFEST_DIR}/clusterrole.yaml"
if [ -f "${CLUSTERROLE}" ]; then
    echo -e "${YELLOW}-> Applying ClusterRole...${NC}"
    kubectl apply -f "${CLUSTERROLE}"
else
    echo -e "${RED}ERROR: ${CLUSTERROLE} not found${NC}"
    exit 1
fi

# Apply clusterrolebinding.yaml
CLUSTERROLEBINDING="${SYSTEM_MANIFEST_DIR}/clusterrolebinding.yaml"
if [ -f "${CLUSTERROLEBINDING}" ]; then
    echo -e "${YELLOW}-> Applying ClusterRoleBinding...${NC}"
    kubectl apply -f "${CLUSTERROLEBINDING}"
else
    echo -e "${RED}ERROR: ${CLUSTERROLEBINDING} not found${NC}"
    exit 1
fi

echo -e "${GREEN}System RBAC applied successfully${NC}"