#!/bin/bash
USERNAME=$1

# Test: Can create StatefulSet in mc-{username}? (should succeed)
# Test: Can create StatefulSet in mc-other? (should fail)
# Test: Can list namespaces? (should succeed)
# Test: Can list services across namespaces? (should succeed)
# Test: Can delete namespace? (should fail)