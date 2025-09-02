REGISTRY ?= ghcr.io
USERNAME ?= sergelogvinov
OCIREPO ?= $(REGISTRY)/$(USERNAME)
HELMREPO ?= $(REGISTRY)/$(USERNAME)/charts
PLATFORM ?= linux/arm64,linux/amd64
PUSH ?= false

SHA ?= $(shell git describe --match=none --always --abbrev=7 --dirty)
TAG ?= $(shell git describe --tag --always --match v[0-9]\*)
GO_LDFLAGS := -ldflags "-w -s -X main.version=$(TAG) -X main.commit=$(SHA)"

OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)
ARCHS = amd64 arm64

BUILD_ARGS := --platform=$(PLATFORM)
ifeq ($(PUSH),true)
BUILD_ARGS += --push=$(PUSH)
BUILD_ARGS += --output type=image,annotation-index.org.opencontainers.image.source="https://github.com/$(USERNAME)/node-labels-exporter",annotation-index.org.opencontainers.image.description="Node labels exporter"
else
BUILD_ARGS += --output type=docker
endif

COSING_ARGS ?=

############

# Help Menu

define HELP_MENU_HEADER
# Getting Started

To build this project, you must have the following installed:

- git
- make
- golang 1.20+
- golangci-lint

endef

export HELP_MENU_HEADER

help: ## This help menu
	@echo "$$HELP_MENU_HEADER"
	@grep -E '^[a-zA-Z0-9%_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

############
#
# Build Abstractions
#

build-all-archs:
	@for arch in $(ARCHS); do $(MAKE) ARCH=$${arch} build ; done

.PHONY: clean
clean: ## Clean
	rm -rf bin .cache

.PHONY: tools
tools:
	go install github.com/google/go-licenses@latest

build-%:
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build $(GO_LDFLAGS) \
		-o bin/$*-$(ARCH) ./cmd/$*

.PHONY: build
build: build-node-labels-exporter ## Build

.PHONY: run
run: build-node-labels-exporter ## Run
	./bin/node-labels-exporter-$(ARCH) --metrics-endpoint=:8080

.PHONY: lint
lint: ## Lint Code
	golangci-lint run --config .golangci.yml

.PHONY: unit
unit: ## Unit Tests
	go test -tags=unit $(shell go list ./...) $(TESTARGS)

.PHONY: licenses
licenses:
	go-licenses check ./... --disallowed_types=forbidden,restricted,reciprocal,unknown

.PHONY: conformance
conformance: ## Conformance
	docker run --rm -it -v $(PWD):/src -w /src ghcr.io/siderolabs/conform:v0.1.0-alpha.30 enforce

############

.PHONY: helm-unit
helm-unit: ## Helm Unit Tests
	@helm lint charts/node-labels-exporter
	@helm template -f charts/node-labels-exporter/ci/values.yaml node-labels-exporter charts/node-labels-exporter >/dev/null

.PHONY: helm-login
helm-login: ## Helm Login
	@echo "${HELM_TOKEN}" | helm registry login $(REGISTRY) --username $(USERNAME) --password-stdin

.PHONY: helm-release
helm-release: ## Helm Release
	@rm -rf dist/
	@helm package charts/node-labels-exporter -d dist
	@helm push dist/node-labels-exporter-*.tgz oci://$(HELMREPO) 2>&1 | tee dist/.digest
	@cosign sign --yes $(COSING_ARGS) $(HELMREPO)/node-labels-exporter@$$(cat dist/.digest | awk -F "[, ]+" '/Digest/{print $$NF}')

############

.PHONY: docs
docs:
	yq -i '.appVersion = "$(TAG)"' charts/node-labels-exporter/Chart.yaml
	helm template -n kube-system node-labels-exporter \
		-f charts/node-labels-exporter/values.edge.yaml \
		charts/node-labels-exporter > docs/deploy/node-labels-exporter.yml
	helm template -n kube-system node-labels-exporter \
		--set-string image.tag=$(TAG) \
		charts/node-labels-exporter > docs/deploy/node-labels-exporter-release.yml
	helm-docs --sort-values-order=file charts/node-labels-exporter

############
#
# Docker Abstractions
#

.PHONY: docker-init
docker-init:
	docker run --rm --privileged multiarch/qemu-user-static:register --reset

	docker context create multiarch ||:
	docker buildx create --name multiarch --driver docker-container --use ||:
	docker context use multiarch
	docker buildx inspect --bootstrap multiarch

image-%:
	docker buildx build $(BUILD_ARGS) \
		--build-arg TAG=$(TAG) \
		--build-arg SHA=$(SHA) \
		-t $(OCIREPO)/$*:$(TAG) \
		--target $* \
		-f Dockerfile .

.PHONY: images-checks
images-checks: images
	trivy image --exit-code 1 --ignore-unfixed --severity HIGH,CRITICAL --no-progress $(OCIREPO)/node-labels-exporter:$(TAG)

.PHONY: images-cosign
images-cosign:
	@cosign sign --yes $(COSING_ARGS) --recursive $(OCIREPO)/node-labels-exporter:$(TAG)

.PHONY: images
images: image-node-labels-exporter ## Build images
