# Application version encoded in all the binaries.
VERSION = 0.0.0

# Base go module name.
MODULE := $(shell cat go.mod | grep -m1 module | awk '{print $$2}')

# Git revision.
REVISION := $(shell git rev-parse HEAD)

# Commands to build, the first lot are architecture agnostic and will be built
# for your host's architecture.  The latter are going to run in Kubernetes, so
# want to be amd64.
CONTROLLERS = \
  unikorn-organization-manager \
  unikorn-project-manager \
  unikorn-control-plane-manager \
  unikorn-cluster-manager \
  unikorn-server \
  unikorn-monitor

# Release will do cross compliation of all images for the 'all' target.
# Note we aren't fucking about with docker here because that opens up a
# whole can of worms to do with caching modules and pisses on performance,
# primarily making me rage.  For image creation, this, by necessity,
# REQUIRES multiarch images to be pushed to a remote registry because
# Docker apparently cannot support this after some 3 years...  So don't
# run that target locally when compiling in release mode.
ifdef RELEASE
CONTROLLER_ARCH := amd64 arm64
BUILDX_OUTPUT := --push
else
CONTROLLER_ARCH := $(shell go env GOARCH)
BUILDX_OUTPUT := --load
endif

# Calculate the platform list to pass to docker buildx.
BUILDX_PLATFORMS := $(shell echo $(patsubst %,linux/%,$(CONTROLLER_ARCH)) | sed 's/ /,/g')

# Some constants to describe the repository.
BINDIR = bin
CMDDIR = cmd
SRCDIR = src
GENDIR = generated
CRDDIR = charts/unikorn/crds
SRVBASE = pkg/server
SRVSCHEMA = $(SRVBASE)/openapi/server.spec.yaml
SRVGENPKG = generated
SRVGENDIR = $(SRVBASE)/$(SRVGENPKG)

# Where to install things.
PREFIX = $(HOME)/bin

# List of binaries to build.
CONTROLLER_BINARIES := $(foreach arch,$(CONTROLLER_ARCH),$(foreach ctrl,$(CONTROLLERS),$(BINDIR)/$(arch)-linux-gnu/$(ctrl)))

# List of sources to trigger a build.
# TODO: Bazel may be quicker, but it's a massive hog, and a pain in the arse.
SOURCES := $(shell find . -type f -name *.go)

SERVER_COMPONENT_SOURCES := $(patsubst %,$(SRVGENDIR)/%,$(SERVER_COMPONENTS))

# Source files defining custom resource APIs
APISRC = $(shell find pkg/apis -name [^z]*.go -type f)

# Some bits about go.
GOPATH := $(shell go env GOPATH)
GOBIN := $(if $(shell go env GOBIN),$(shell go env GOBIN),$(GOPATH)/bin)

# Common linker flags.
FLAGS=-trimpath -ldflags '-X $(MODULE)/pkg/constants.Version=$(VERSION) -X $(MODULE)/pkg/constants.Revision=$(REVISION)'

# Defines the linter version.
LINT_VERSION=v1.54.2

# Defines the version of the CRD generation tools to use.
CONTROLLER_TOOLS_VERSION=v0.12.1

# Defines the version of code generator tools to use.
# This should be kept in sync with the Kubenetes library versions defined in go.mod.
CODEGEN_VERSION=v0.27.3

OPENAPI_CODEGEN_VERSION=v1.12.4

MOCKGEN_VERSION=v0.3.0

# This is the base directory to generate kubernetes API primitives from e.g.
# clients and CRDs.
GENAPIBASE = github.com/unikorn-cloud/unikorn/pkg/apis

# This is the list of APIs to generate clients for.
GENAPIS = $(GENAPIBASE)/unikorn/v1alpha1

# These are generic arguments that need to be passed to client generation.
GENARGS = --go-header-file hack/boilerplate.go.txt --output-base ../../..

# This defines how docker containers are tagged.
DOCKER_ORG = ghcr.io/unikorn-cloud

# Main target, builds all binaries.
.PHONY: all
all: $(CONTROLLER_BINARIES) $(CRDDIR)

# Create a binary output directory, this should be an order-only prerequisite.
$(BINDIR) $(BINDIR)/amd64-linux-gnu $(BINDIR)/arm64-linux-gnu:
	mkdir -p $@

# Create a binary from a command.
$(BINDIR)/%: $(SOURCES) $(GENDIR) $(SRVGENDIR) | $(BINDIR)
	CGO_ENABLED=0 go build $(FLAGS) -o $@ $(CMDDIR)/$*/main.go

$(BINDIR)/amd64-linux-gnu/%: $(SOURCES) $(GENDIR) $(SRVGENDIR) | $(BINDIR)/amd64-linux-gnu
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(FLAGS) -o $@ $(CMDDIR)/$*/main.go

$(BINDIR)/arm64-linux-gnu/%: $(SOURCES) $(GENDIR) | $(BINDIR)/arm64-linux-gnu
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(FLAGS) -o $@ $(CMDDIR)/$*/main.go

# TODO: we may wamt to consider porting the rest of the CRD and client generation
# stuff over... that said, we don't need the clients really do we, controller-runtime
# does all the magic for us.
.PHONY: generate
generate:
	@go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
	go generate ./...

# Create container images.  Use buildkit here, as it's the future, and it does
# good things, like per file .dockerignores and all that jazz.
.PHONY: images
images: $(CONTROLLER_BINARIES)
	if [ -n "$(RELEASE)" ]; then docker buildx create --name unikorn --use; fi
	for image in ${CONTROLLERS}; do docker buildx build --platform $(BUILDX_PLATFORMS) $(BUILDX_OUTPUT) -f docker/$${image}/Dockerfile -t ${DOCKER_ORG}/$${image}:${VERSION} .; done;
	if [ -n "$(RELEASE)" ]; then docker buildx rm unikorn; fi

# Purely lazy command that builds and pushes to docker hub.
.PHONY: images-push
images-push: images
	for image in ${CONTROLLERS}; do docker push ${DOCKER_ORG}/$${image}:${VERSION}; done

.PHONY: images-kind-load
images-kind-load: images
	for image in ${CONTROLLERS}; do kind load docker-image ${DOCKER_ORG}/$${image}:${VERSION}; done

.PHONY: test-unit
test-unit:
	go test -coverpkg ./... -coverprofile cover.out ./...
	go tool cover -html cover.out -o cover.html

# Build a binary and install it.
$(PREFIX)/%: $(BINDIR)/%
	install -m 750 $< $@

# Create any CRDs defined into the target directory.
$(CRDDIR): $(APISRC)
	@mkdir -p $@
	@go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)
	$(GOBIN)/controller-gen crd:crdVersions=v1 paths=./pkg/apis/unikorn/... output:dir=$@
	@touch $(CRDDIR)

# Generate a clientset to interact with our custom resources.
$(GENDIR): $(APISRC)
	@go install k8s.io/code-generator/cmd/deepcopy-gen@$(CODEGEN_VERSION)
	$(GOBIN)/deepcopy-gen --input-dirs $(GENAPIS) -O zz_generated.deepcopy --bounding-dirs $(GENAPIBASE) $(GENARGS)
	@touch $@

# Generate the server schema, types and router boilerplate.
$(SRVGENDIR): $(SRVSCHEMA)
	@mkdir -p $@
	@go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@$(OPENAPI_CODEGEN_VERSION)
	oapi-codegen -generate spec -package $(SRVGENPKG) $< > $(SRVGENDIR)/schema.go
	oapi-codegen -generate types -package $(SRVGENPKG) $< > $(SRVGENDIR)/types.go
	oapi-codegen -generate chi-server -package $(SRVGENPKG) $< > $(SRVGENDIR)/router.go
	oapi-codegen -generate client -package $(SRVGENPKG) $< > $(SRVGENDIR)/client.go
	@touch $@

# When checking out, the files timestamps are pretty much random, and make cause
# spurious rebuilds of generated content.  Call this to prevent that.
.PHONY: touch
touch:
	touch $(CRDDIR) $(GENDIR) pkg/apis/unikorn/v1alpha1/zz_generated.deepcopy.go

# Perform linting.
# This must pass or you will be denied by CI.
.PHOMY: lint
lint: $(GENDIR)
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(LINT_VERSION)
	$(GOBIN)/golangci-lint run ./...
	helm lint --strict charts/unikorn

# Validate the server OpenAPI schema is legit.
.PHONY: validate
validate: $(SRVGENDIR)
	go run ./hack/validate_openapi

# Validate the docs can be generated without fail.
.PHONY: validate-docs
validate-docs: $(SRVGENDIR)
	go run github.com/unikorn-cloud/core/hack/docs --dry-run

# Perform license checking.
# This must pass or you will be denied by CI.
.PHONY: license
license:
	go run github.com/unikorn-cloud/core/hack/check_license
