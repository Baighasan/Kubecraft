#!/bin/bash

# Exit on errors
set -euo pipefail

USERNAME=$1

# Validate input
if [ -z "${USERNAME}" ]; then
    echo -e "${RED}ERROR: Username is required${NC}"
    echo "Usage: $0 <username>"
    exit 1
fi

# Get script directory for relative paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
USER_MANIFEST_DIR="${SCRIPT_DIR}/../manifests/user-templates"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Applying user manifests...${NC}"

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}ERROR: kubectl not found. Please install kubectl.${NC}"
    exit 1
fi

# Check if manifest directory exists
if [ ! -d "${USER_MANIFEST_DIR}" ]; then
    echo -e "${RED}ERROR: Manifest directory not found: ${USER_MANIFEST_DIR}${NC}"
    exit 1
fi

# Validate username format (Kubernetes DNS label compatible)
# - Length: 3-16 characters
# - Characters: a-z, 0-9 only
# - Must start and end with alphanumeric character
if ! [[ "${USERNAME}" =~ ^[a-z0-9]{3,16}$ ]]; then
    echo -e "${RED}ERROR: Invalid username '${USERNAME}'${NC}"
    echo "Username must:"
    echo "  • Be 3-16 characters long"
    echo "  • Contain only lowercase letters (a-z) and numbers (0-9)"
    echo "  • No special characters"
    exit 1
fi

# Check if namespace already exists
if kubectl get namespace "mc-${USERNAME}" &> /dev/null; then
    echo -e "${RED}ERROR: User '${USERNAME}' already exists (namespace mc-${USERNAME} found)${NC}"
    exit 1
fi

# Replace {username} placeholders using sed & apply user manifests in order
for manifest in namespace resourcequota serviceaccount role rolebinding; do
    manifest_file="${USER_MANIFEST_DIR}/${manifest}.yaml"

    if [ ! -f "${manifest_file}" ]; then
        echo -e "${RED}ERROR: ${manifest_file} not found${NC}"
        exit 1
    fi

    echo -e "${YELLOW}-> Applying ${manifest}...${NC}"
    sed "s/{username}/${USERNAME}/g" "${manifest_file}" | kubectl apply -f -
done

# Update ClusterRoleBinding to grant capacity-check permissions
echo -e "${YELLOW}-> Adding user to ClusterRoleBinding...${NC}"

# Check if subjects array is empty/null (omitted field returns empty string)
SUBJECTS_OUTPUT=$(kubectl get clusterrolebinding kc-users-capacity-check -o jsonpath='{.subjects}')

if [ -z "$SUBJECTS_OUTPUT" ]; then
  # First user - initialize the subjects array
  kubectl patch clusterrolebinding kc-users-capacity-check --type='json' -p='[
    {
      "op": "replace",
      "path": "/subjects",
      "value": [{
        "kind": "ServiceAccount",
        "name": "'"${USERNAME}"'",
        "namespace": "mc-'"${USERNAME}"'"
      }]
    }
  ]'
else
  # Append to existing subjects array
  kubectl patch clusterrolebinding kc-users-capacity-check --type='json' -p='[
    {
      "op": "add",
      "path": "/subjects/-",
      "value": {
        "kind": "ServiceAccount",
        "name": "'"${USERNAME}"'",
        "namespace": "mc-'"${USERNAME}"'"
      }
    }
  ]'
fi

# Register user (implement after auth is complete)

echo -e "${GREEN}User successfully created!${NC}"