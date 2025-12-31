#!/bin/bash
USERNAME=$1

# Delete namespace (cascades to all resources inside)
# Remove kubeconfig-{username}.yaml
# (Optional) Remove user from ClusterRoleBinding