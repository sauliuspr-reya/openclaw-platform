# OpenClaw Agent Image Build Makefile

REGISTRY ?= ghcr.io/openclaw
VERSION ?= latest

PROFILES = base minimal python-ml devops web data security

.PHONY: all build push clean $(PROFILES)

all: build

# Build all images
build: $(PROFILES)

# Build individual profiles
base:
	docker build -t $(REGISTRY)/agent:base -f base/Dockerfile base/

minimal: base
	docker build -t $(REGISTRY)/agent:minimal -t $(REGISTRY)/agent:$(VERSION) -f minimal/Dockerfile minimal/

python-ml: base
	docker build -t $(REGISTRY)/agent:python-ml -f python-ml/Dockerfile python-ml/

devops: base
	docker build -t $(REGISTRY)/agent:devops -f devops/Dockerfile devops/

web: base
	docker build -t $(REGISTRY)/agent:web -f web/Dockerfile web/

data: base
	docker build -t $(REGISTRY)/agent:data -f data/Dockerfile data/

security: base
	docker build -t $(REGISTRY)/agent:security -f security/Dockerfile security/

# Push all images
push:
	@for profile in $(PROFILES); do \
		docker push $(REGISTRY)/agent:$$profile; \
	done
	docker push $(REGISTRY)/agent:$(VERSION)

# Push individual profile
push-%:
	docker push $(REGISTRY)/agent:$*

# Clean local images
clean:
	@for profile in $(PROFILES); do \
		docker rmi $(REGISTRY)/agent:$$profile 2>/dev/null || true; \
	done
	docker rmi $(REGISTRY)/agent:$(VERSION) 2>/dev/null || true

# Multi-arch build (for production)
build-multiarch:
	@for profile in $(PROFILES); do \
		docker buildx build --platform linux/amd64,linux/arm64 \
			-t $(REGISTRY)/agent:$$profile \
			-f $$profile/Dockerfile \
			$$profile/ --push; \
	done

# List image sizes
sizes:
	@echo "Image sizes:"
	@for profile in $(PROFILES); do \
		size=$$(docker images $(REGISTRY)/agent:$$profile --format "{{.Size}}"); \
		printf "  %-15s %s\n" "$$profile:" "$$size"; \
	done

# Test images
test:
	@for profile in $(PROFILES); do \
		echo "Testing $(REGISTRY)/agent:$$profile..."; \
		docker run --rm $(REGISTRY)/agent:$$profile echo "$$profile OK"; \
	done

# Generate manifest of installed tools per profile
manifest:
	@echo "Generating tool manifests..."
	@mkdir -p manifests
	@for profile in $(PROFILES); do \
		docker run --rm $(REGISTRY)/agent:$$profile printenv > manifests/$$profile-env.txt 2>/dev/null || true; \
		docker run --rm $(REGISTRY)/agent:$$profile which python3 pip node npm kubectl helm terraform 2>/dev/null | head -20 > manifests/$$profile-tools.txt || true; \
	done
	@echo "Manifests written to manifests/"
