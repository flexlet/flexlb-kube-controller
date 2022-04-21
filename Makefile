ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

CONTROLLER_GEN = $(GOBIN)/controller-gen

.PHONY: all
all: build

##@ Development

.PHONY: manifests
manifests: 
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: 
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

##@ Build

.PHONY: build
build: generate
	go build -o bin/flexlb-kube-controller main.go

.PHONY: run
run: manifests generate
	go run ./main.go

##@ Deployment

.PHONY: install
install: manifests
	kubectl apply -f config/crd/bases

.PHONY: uninstall
uninstall: manifests
	kubectl delete -f config/crd/bases
