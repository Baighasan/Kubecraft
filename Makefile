BINARY = kubecraft
MODULE = github.com/baighasan/kubecraft/internal/config

# Dev defaults
DEV_ENDPOINT ?= 0.0.0.0:43835
DEV_NODE_ADDRESS ?= localhost
DEV_SERVER_IMAGE_TAG ?= dev

# Prod defaults (update with your EC2 IP)
PROD_ENDPOINT ?= CHANGEME:6443
PROD_NODE_ADDRESS ?= CHANGEME
PROD_SERVER_IMAGE_TAG ?= latest

LDFLAGS_DEV = -X $(MODULE).ClusterEndpoint=$(DEV_ENDPOINT) -X $(MODULE).NodeAddress=$(DEV_NODE_ADDRESS) -X $(MODULE).TLSInsecure=true -X $(MODULE).ServerImage=ghcr.io/baighasan/kubecraft-minecraft:$(DEV_SERVER_IMAGE_TAG)
LDFLAGS_PROD = -X $(MODULE).ClusterEndpoint=$(PROD_ENDPOINT) -X $(MODULE).NodeAddress=$(PROD_NODE_ADDRESS) -X $(MODULE).TLSInsecure=false -X $(MODULE).ServerImage=ghcr.io/baighasan/kubecraft-minecraft:$(PROD_SERVER_IMAGE_TAG)

.PHONY: build-dev build-prod test clean cluster-up cluster-down cluster-setup

build-dev:
	go build -ldflags "$(LDFLAGS_DEV)" -o $(BINARY) ./cmd/kubecraft

build-prod:
	go build -ldflags "$(LDFLAGS_PROD)" -o $(BINARY) ./cmd/kubecraft

test:
	go test -race ./internal/config/... ./internal/registration/... ./internal/cli ./internal/cli/server

clean:
	rm -f $(BINARY)

cluster-up:
	k3d cluster create kubecraft-dev --port "30000-30099:30000-30099@server:0"

cluster-setup:
	helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane

cluster-down:
	k3d cluster delete kubecraft-dev
