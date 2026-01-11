#!/bin/bash

# Exit on errors
set -euo pipefail

# Validate username argument
if [ $# -eq 0 ]; then
    echo "Usage: $0 <username>"
    echo "Example: $0 alice"
    exit 1
fi

USERNAME=$1

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}ERROR: kubectl not found. Please install kubectl.${NC}"
    exit 1
fi

# Check if namespace does not exists
if ! kubectl get namespace "mc-${USERNAME}" &> /dev/null; then
    echo -e "${RED}ERROR: User '${USERNAME}' does not exist (namespace mc-${USERNAME} not found)${NC}"
    exit 1
fi

# Delete namespace (cascades to all resources inside)
echo -e "${YELLOW}Deleting namespace mc-${USERNAME}${NC}"
kubectl delete namespace "mc-${USERNAME}"

# Remove user from ClusterRoleBinding
# Get all subject names and namespaces using kubectl jsonpath
SUBJECTS=$(kubectl get clusterrolebinding kc-users-capacity-check -o jsonpath='{range .subjects[*]}{.name},{.namespace}{"\n"}{end}')

# Find the index of the subject matching our username and namespace
SUBJECT_INDEX=-1
INDEX=0
while IFS=',' read -r name namespace; do
    if [ "$name" == "$USERNAME" ] && [ "$namespace" == "mc-$USERNAME" ]; then
        SUBJECT_INDEX=$INDEX
        break
    fi
    ((INDEX++))
done <<< "$SUBJECTS"

if [ "$SUBJECT_INDEX" -eq -1 ]; then
    echo -e "${YELLOW}WARNING: User '${USERNAME}' not found in ClusterRoleBinding (already removed or never added)${NC}"
else
    echo -e "${YELLOW}Removing user from ClusterRoleBinding...${NC}"
    if kubectl patch clusterrolebinding kc-users-capacity-check --type='json' -p="[
      {
        \"op\": \"remove\",
        \"path\": \"/subjects/${SUBJECT_INDEX}\"
      }
    ]"; then
        echo -e "${GREEN}✓ Removed user from ClusterRoleBinding${NC}"
    else
        echo -e "${RED}ERROR: Failed to remove user from ClusterRoleBinding${NC}"
        exit 1
    fi
fi

echo ""
echo -e "${GREEN}✅ User '${USERNAME}' has been completely deleted${NC}"