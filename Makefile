# OpenClaw Platform Makefile

# Image URLs
REGISTRY ?= ghcr.io/openclaw
OPERATOR_IMG ?= $(REGISTRY)/operator:latest
VERSION ?= 0.1.0

# Go settings
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Kubernetes tools
KUBECTL ?= kubectl
KUSTOMIZE ?= kustomize

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: test
test: ## Run tests
	go test ./... -coverprofile cover.out

.PHONY: generate
generate: ## Generate code (CRD, DeepCopy, etc.)
	go generate ./...

##@ Build

.PHONY: build
build: fmt vet ## Build operator binary
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/manager cmd/operator/main.go

.PHONY: docker-build
docker-build: ## Build operator Docker image
	docker build -t $(OPERATOR_IMG) -f Dockerfile .

.PHONY: docker-push
docker-push: ## Push operator Docker image
	docker push $(OPERATOR_IMG)

.PHONY: docker-build-images
docker-build-images: ## Build all agent images
	$(MAKE) -C deploy/images build

.PHONY: docker-push-images
docker-push-images: ## Push all agent images
	$(MAKE) -C deploy/images push

##@ Deployment

.PHONY: install-crds
install-crds: ## Install CRDs into the cluster
	$(KUBECTL) apply -f config/crd/

.PHONY: uninstall-crds
uninstall-crds: ## Uninstall CRDs from the cluster
	$(KUBECTL) delete -f config/crd/

.PHONY: deploy
deploy: install-crds ## Deploy operator to the cluster
	$(KUBECTL) apply -f config/rbac/
	$(KUBECTL) apply -f config/manager/

.PHONY: undeploy
undeploy: ## Undeploy operator from the cluster
	$(KUBECTL) delete -f config/manager/
	$(KUBECTL) delete -f config/rbac/

.PHONY: deploy-nats
deploy-nats: ## Deploy NATS JetStream
	$(KUBECTL) apply -f deploy/nats/nats-deployment.yaml

.PHONY: init-buckets
init-buckets: ## Initialize NATS memory buckets
	$(KUBECTL) exec -n openclaw deploy/nats-box -- /bin/sh -c "$$(cat deploy/nats/init-buckets.sh)"

.PHONY: deploy-examples
deploy-examples: ## Deploy example agents
	$(KUBECTL) apply -f examples/

##@ Local Development

.PHONY: run
run: ## Run operator locally (against current kubeconfig cluster)
	go run cmd/operator/main.go

.PHONY: kind-create
kind-create: ## Create a local Kind cluster
	kind create cluster --name openclaw --config hack/kind-config.yaml

.PHONY: kind-delete
kind-delete: ## Delete the local Kind cluster
	kind delete cluster --name openclaw

.PHONY: kind-load
kind-load: docker-build ## Load operator image into Kind
	kind load docker-image $(OPERATOR_IMG) --name openclaw

##@ Utilities

.PHONY: manifests
manifests: ## Generate manifests (CRD, RBAC, etc.)
	controller-gen rbac:roleName=openclaw-operator-role crd webhook paths="./..." output:crd:artifacts:config=config/crd output:rbac:artifacts:config=config/rbac

.PHONY: deps
deps: ## Download dependencies
	go mod download
	go mod tidy

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f cover.out

.PHONY: version
version: ## Print version
	@echo $(VERSION)
