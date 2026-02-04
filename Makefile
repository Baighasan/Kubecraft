BINARY = kubecraft
MODULE = github.com/baighasan/kubecraft/internal/config

# Dev defaults
DEV_ENDPOINT ?= 0.0.0.0:43835
DEV_NODE_ADDRESS ?= localhost

# Prod defaults (update with your EC2 IP)
PROD_ENDPOINT ?= CHANGEME:6443
PROD_NODE_ADDRESS ?= CHANGEME

LDFLAGS_DEV = -X $(MODULE).ClusterEndpoint=$(DEV_ENDPOINT) -X $(MODULE).NodeAddress=$(DEV_NODE_ADDRESS) -X $(MODULE).TLSInsecure=true
LDFLAGS_PROD = -X $(MODULE).ClusterEndpoint=$(PROD_ENDPOINT) -X $(MODULE).NodeAddress=$(PROD_NODE_ADDRESS) -X $(MODULE).TLSInsecure=false

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
	kubectl apply -f manifests/system-templates/
	kubectl apply -f manifests/registration-templates/

cluster-down:
	k3d cluster delete kubecraft-dev
