PROJECT_FULL_NAME := control-plane-operator

# Image URL to use all building/pushing image targets
IMG_VERSION ?= dev
IMG_BASE ?= $(PROJECT_FULL_NAME)
IMG ?= $(IMG_BASE):$(IMG_VERSION)
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
# Pick from https://storage.googleapis.com/kubebuilder-tools
ENVTEST_K8S_VERSION = 1.30.0

export UUT_IMAGES = {"cloud-orchestration/control-plane-operator":"$(IMG)"}
SET_BASE_DIR := $(eval BASE_DIR=$(shell git rev-parse --show-toplevel))
GENERATED_DIR := ${BASE_DIR}/hack/.generated

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

GOOS ?= linux
GOARCH ?= $(shell go env GOARCH)
CONTAINER_PLATFORM ?= $(GOOS)/$(GOARCH)

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) crd paths="./..." output:crd:artifacts:config=cmd/embedded/crds

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out
	go tool cover --html=cover.out -o cover.html
	go tool cover -func cover.out | tail -n 1

.PHONY: lint 
lint: ## Run golangci-lint to lint code
	golangci-lint run ./... --timeout=15m

.PHONY: tidy
tidy: 
	go mod tidy -e

.PHONY: verify
verify: lint goimports vet 

.PHONY: localbin
localbin:
	@test -d $(LOCALBIN) || mkdir -p $(LOCALBIN)

FORMATTER_VERSION ?= v0.26.0

.PHONY: goimports
goimports: localbin ## Download goimports locally if necessary. If wrong version is installed, it will be overwritten.
	@test -s $(FORMATTER) && test -s hack/goimports_version && cat hack/goimports_version | grep -q $(FORMATTER_VERSION) || \
	( echo "Installing goimports $(FORMATTER_VERSION) ..."; \
	GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@$(FORMATTER_VERSION) && \
	echo $(FORMATTER_VERSION) > hack/goimports_version )

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go start

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -a -o bin/manager-linux.$(GOARCH) cmd/main.go
	$(CONTAINER_TOOL) build --platform $(CONTAINER_PLATFORM) -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm project-v3-builder
	rm Dockerfile.cross

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: pull-secrets
pull-secrets:
	@echo "Creating Pull Secret(s)"
	mkdir -p ${GENERATED_DIR}
	${BASE_DIR}/hack/create-token-opt.sh

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOTESTSUM ?= $(LOCALBIN)/gotestsum
KIND ?= kind # fix this to use tools

## Tool Versions
KUSTOMIZE_VERSION ?= v5.1.1
CONTROLLER_TOOLS_VERSION ?= v0.16.4

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) GO111MODULE=on go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: gotestsum
gotestsum: $(GOTESTSUM) ## Download gotestsum locally if necessary.
$(GOTESTSUM): $(LOCALBIN)
	test -s $(LOCALBIN)/gotestsum || GOBIN=$(LOCALBIN) go install gotest.tools/gotestsum@latest

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

### ------------------------------------ DEVELOPMENT - LOCAL ------------------------------------ ###
.PHONY: dev-build
dev-build: docker-build
	@echo "Finished building docker image" ${IMG}

.PHONY: dev-base
dev-base: manifests kustomize dev-build dev-clean dev-cluster flux-install flux-secret helm-install-local

.PHONY: dev-cluster
dev-cluster:
	$(KIND) create cluster --name=$(PROJECT_FULL_NAME)-dev
	$(KIND) load docker-image ${IMG} --name=$(PROJECT_FULL_NAME)-dev

.PHONY: dev-local
dev-local:
	$(KIND) create cluster --name=$(PROJECT_FULL_NAME)-dev
	$(MAKE) install
	$(MAKE) flux-install
	$(MAKE) flux-secret

.PHONY: helm-install-local
helm-install-local:
	helm upgrade --create-namespace --namespace co-system --install $(PROJECT_FULL_NAME) charts/$(PROJECT_FULL_NAME)/ -f test/e2e/testdata/values.yaml --set image.repository=$(IMG_BASE) --set image.tag=$(IMG_VERSION) --set image.pullPolicy=Never

.PHONY: dev-clean
dev-clean:
	$(KIND) delete cluster --name=$(PROJECT_FULL_NAME)-dev

.PHONY: dev-run
dev-run:
	## todo: add flag --debug
	go run ./cmd/main.go start


.PHONY: flux-install
flux-install:
	kubectl apply -f https://github.com/fluxcd/flux2/releases/latest/download/install.yaml

### ------------------------------------ HELM ------------------------------------ ###

.PHONY: helm-chart
helm-chart: helm-templates
	OPERATOR_VERSION=$(shell cat VERSION) envsubst < charts/$(PROJECT_FULL_NAME)/Chart.yaml.tpl > charts/$(PROJECT_FULL_NAME)/Chart.yaml
	OPERATOR_VERSION=$(shell cat VERSION) envsubst < charts/$(PROJECT_FULL_NAME)/values.yaml.tpl > charts/$(PROJECT_FULL_NAME)/values.yaml

### ------------------------------------------------------------------------------ ###

# initializes pre-commit hooks using lefthook https://github.com/evilmartians/lefthook
lefthook:
	@lefthook install

### ------------------------------------ E2E - Tests ------------------------------------ ###
.PHONY: e2e
e2e: helm-chart docker-build e2e.prep run-e2e-with-report

.PHONY: run-e2e
run-e2e: docker-build
	go test -v ./... -tags=e2e

.PHONY: run-e2e-with-report
run-e2e-with-report: docker-build run-e2e-with-report-only

.PHONY: e2e.prep
e2e.prep: docker-build pull-secrets
	echo E2E_IMAGES=$$UUT_IMAGES > e2e.env
	echo PULL_SECRET_USER=$(shell cat ${GENERATED_DIR}/artifactory-user) > secret.env
	-echo PULL_SECRET_PASSWORD=$(shell cat ${GENERATED_DIR}/artifactory-bearer-token.json) >> secret.env

run-e2e-with-report-only: gotestsum
	@echo "UUT_IMAGES=$$UUT_IMAGES"
	$(GOTESTSUM) --debug --format standard-verbose --junitfile=integration-tests.xml -- --tags=e2e ./.../e2e -test.v -test.short -short -v -timeout 30m
