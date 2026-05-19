BINARY = kubecraft
MODULE = github.com/baighasan/kubecraft/internal/config

# Image tag override
SERVER_IMAGE_TAG ?= dev

LDFLAGS = -X $(MODULE).ServerImage=ghcr.io/baighasan/kubecraft-minecraft:$(SERVER_IMAGE_TAG)

.PHONY: build test clean cluster-up cluster-down cluster-setup

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/kubecraft

test:
	go test -race ./internal/config/... ./internal/registration/... ./internal/cli ./internal/cli/server

clean:
	rm -f $(BINARY)

cluster-up:
	k3d cluster create kubecraft-dev --api-port 0.0.0.0:6443 --port "30000-30099:30000-30099@server:0"

cluster-setup:
	helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane --set registration.image.tag=dev

cluster-down:
	k3d cluster delete kubecraft-dev
