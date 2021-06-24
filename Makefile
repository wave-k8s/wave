include Makefile.tools
include .env

SHELL := /usr/bin/env bash

GO := go
LINTER := golangci-lint
BINARY := wave
VERSION := $(shell git describe --always --dirty --tags 2>/dev/null || echo "undefined")
ECHO := echo -e

# Image URL to use all building/pushing image targets
IMG ?= quay.io/wave-k8s/wave

.NOTPARALLEL:

.PHONY: all
all: test build

.PHONY: build
build: clean $(BINARY)

.PHONY: clean
clean:
	rm -f $(BINARY)

.PHONY: distclean
distclean: clean
	rm -rf vendor
	rm -rf release

# Generate code
.PHONY: generate
generate: vendor
	@ $(ECHO) "\033[36mGenerating code\033[0m"
	$(GO) generate ./pkg/... ./cmd/...
	@ $(ECHO)

# Verify generated code has been checked in
.PHONY: verify-%
verify-%:
	@ make $*
	@ $(ECHO) "\033[36mVerifying Git Status\033[0m"
	@ if [ "$$(git status -s)" != "" ]; then git --no-pager diff --color; $(ECHO) "\033[31;1mERROR: Git Diff found. Please run \`make $*\` and commit the result.\033[0m"; exit 1; else $(ECHO) "\033[32mVerified $*\033[0m";fi
	@ $(ECHO)

# Run go fmt against code
.PHONY: fmt
fmt:
	$(GO) fmt ./pkg/... ./cmd/...

# Run go vet against code
.PHONY: vet
vet:
	$(GO) vet ./pkg/... ./cmd/...

.PHONY: lint
lint: vendor
	@ $(ECHO) "\033[36mLinting code\033[0m"
	$(LINTER) run --disable-all \
                --exclude-use-default=false \
                --enable=govet \
                --enable=ineffassign \
                --enable=deadcode \
                --enable=golint \
                --enable=goconst \
                --enable=gofmt \
                --enable=goimports \
                --skip-dirs=pkg/client/ \
                --deadline=120s \
                --tests ./...
	@ $(ECHO)

# Run tests
export TEST_ASSET_KUBECTL := $(KUBEBUILDER)/kubectl
export TEST_ASSET_KUBE_APISERVER := $(KUBEBUILDER)/kube-apiserver
export TEST_ASSET_ETCD := $(KUBEBUILDER)/etcd

vendor:
	@ $(ECHO) "\033[36mPuling dependencies\033[0m"
	go get -v github.com/wave-k8s/wave/cmd/manager
	@ $(ECHO)

.PHONY: check
check: fmt lint vet test

.PHONY: test
test: vendor generate manifests
	@ $(ECHO) "\033[36mRunning test suite in Ginkgo\033[0m"
	$(GINKGO) -v -randomizeAllSpecs ./pkg/... ./cmd/... -- -report-dir=$$ARTIFACTS
	@ $(ECHO)

# Build manager binary
$(BINARY): generate fmt vet
	CGO_ENABLED=0 $(GO) build -o $(BINARY) -ldflags="-X main.VERSION=${VERSION}" github.com/wave-k8s/wave/cmd/manager

# Build all arch binaries
release: test docker-build docker-tag docker-push
	mkdir -p release
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.VERSION=${VERSION}" -o release/$(BINARY)-darwin-amd64 github.com/wave-k8s/wave/cmd/manager
	GOOS=linux GOARCH=amd64 go build -ldflags="-X main.VERSION=${VERSION}" -o release/$(BINARY)-linux-amd64 github.com/wave-k8s/wave/cmd/manager
	GOOS=linux GOARCH=arm64 go build -ldflags="-X main.VERSION=${VERSION}" -o release/$(BINARY)-linux-arm64 github.com/wave-k8s/wave/cmd/manager
	GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-X main.VERSION=${VERSION}" -o release/$(BINARY)-linux-armv6 github.com/wave-k8s/wave/cmd/manager
	GOOS=windows GOARCH=amd64 go build -ldflags="-X main.VERSION=${VERSION}" -o release/$(BINARY)-windows-amd64 github.com/wave-k8s/wave/cmd/manager
	$(SHASUM) -a 256 release/$(BINARY)-darwin-amd64 > release/$(BINARY)-darwin-amd64-sha256sum.txt
	$(SHASUM) -a 256 release/$(BINARY)-linux-amd64 > release/$(BINARY)-linux-amd64-sha256sum.txt
	$(SHASUM) -a 256 release/$(BINARY)-linux-arm64 > release/$(BINARY)-linux-arm64-sha256sum.txt
	$(SHASUM) -a 256 release/$(BINARY)-linux-armv6 > release/$(BINARY)-linux-armv6-sha256sum.txt
	$(SHASUM) -a 256 release/$(BINARY)-windows-amd64 > release/$(BINARY)-windows-amd64-sha256sum.txt
	$(TAR) -czvf release/$(BINARY)-$(VERSION).darwin-amd64.$(GOVERSION).tar.gz release/$(BINARY)-darwin-amd64
	$(TAR) -czvf release/$(BINARY)-$(VERSION).linux-amd64.$(GOVERSION).tar.gz release/$(BINARY)-linux-amd64
	$(TAR) -czvf release/$(BINARY)-$(VERSION).linux-arm64.$(GOVERSION).tar.gz release/$(BINARY)-linux-arm64
	$(TAR) -czvf release/$(BINARY)-$(VERSION).linux-armv6.$(GOVERSION).tar.gz release/$(BINARY)-linux-armv6
	$(TAR) -czvf release/$(BINARY)-$(VERSION).windows-amd64.$(GOVERSION).tar.gz release/$(BINARY)-windows-amd64

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: generate fmt vet
	$(GO) run ./cmd/manager/main.go

# Install CRDs into a cluster
.PHONY: install
install: manifests
	$(KUBECTL) apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
.PHONY: deploy
deploy: manifests
	$(KUBECTL) apply -f config/crds
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
.PHONY: manifests
manifests: vendor
	@ $(ECHO) "\033[36mGenerating manifests\033[0m"
	$(GO) run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all
	@ $(ECHO)

# Build the docker image
.PHONY: docker-build
docker-build:
	docker build --build-arg VERSION=${VERSION} -t ${IMG}:${VERSION} .
	@$(ECHO) "\033[36mBuilt $(IMG):$(VERSION)\033[0m"

TAGS ?= latest
.PHONY: docker-tag
docker-tag:
	@IFS=","; tags=${TAGS}; for tag in $${tags}; do docker tag ${IMG}:${VERSION} ${IMG}:$${tag}; $(ECHO) "\033[36mTagged $(IMG):$(VERSION) as $${tag}\033[0m"; done

# Push the docker image
PUSH_TAGS ?= ${VERSION},latest
.PHONY: docker-push
docker-push:
	@IFS=","; tags=${PUSH_TAGS}; for tag in $${tags}; do docker push ${IMG}:$${tag}; $(ECHO) "\033[36mPushed $(IMG):$${tag}\033[0m"; done
