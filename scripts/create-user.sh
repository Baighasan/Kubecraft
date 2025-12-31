#!/bin/bash
USERNAME=$1

# Validate input
# Replace {username} placeholders using sed
# Apply namespace.yaml, resourcequota.yaml, role.yaml, rolebinding.yaml, serviceaccount.yaml
# Extract ServiceAccount token
# Generate kubeconfig-{username}.yaml file