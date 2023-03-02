.DEFAULT_GOAL := build-server

##################################
# git
##################################
GIT_URL ?= $(shell git remote get-url --push origin)
GIT_COMMIT ?= $(shell git rev-parse HEAD)
#GIT_SHORT_COMMIT := $(shell git rev-parse --short HEAD)
TIMESTAMP := $(shell date '+%Y-%m-%d_%I:%M:%S%p')
REGION ?= us-east-2
ECR_PUBLIC ?= public.ecr.aws
IMAGE_REGISTRY ?= <ACCOUNT>.dkr.ecr.<REGION>.amazonaws.com
INIT_IMAGE_REPO ?= notary-admission-init
INIT_DOCKERFILE ?= Dockerfile.init
SERVER_IMAGE_REPO ?= notary-admission
SERVER_DOCKERFILE ?= Dockerfile.server
NO_CACHE ?= true
GIT_COMMIT_IN ?=
GIT_URL_IN ?=
GO_MOD_PATH ?= notary-admission
PLATFORM ?= linux/amd64

ifeq ($(strip $(GIT_COMMIT)),)
GIT_COMMIT := $(GIT_COMMIT_IN)
endif

ifeq ($(strip $(GIT_URL)),)
GIT_URL := $(GIT_URL_IN)
endif

VERSION_HASH := $(shell echo $(GIT_COMMIT)|cut -c 1-10)
# $(info [$(VERSION_HASH)])
VERSION_FROM_FILE ?= $(shell head -n 1 version)
VERSION ?=

ifeq ($(strip $(VERSION_HASH)),)
VERSION := $(VERSION_FROM_FILE)
else
VERSION := $(VERSION_FROM_FILE)-$(VERSION_HASH)
endif

.PHONY: build-server build-init login logout push-server push-init pull meta clean compile-server compile-init init check test run help

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build and deploy

build-server:	meta ## Build container with Docker buildx, based on PLATFORM argument (default linux/amd64)
	docker buildx build -f Dockerfile.server \
		--load \
		--platform $(PLATFORM) \
		--tag $(IMAGE_REGISTRY)/$(SERVER_IMAGE_REPO):$(VERSION) . --no-cache=$(NO_CACHE)

build-init:	meta ## Build container with Docker buildx, based on PLATFORM argument (default linux/amd64)
	docker buildx build -f Dockerfile.init \
		--load \
		--platform $(PLATFORM) \
		--tag $(IMAGE_REGISTRY)/$(INIT_IMAGE_REPO):$(VERSION) . --no-cache=$(NO_CACHE)

login:	## Login to remote image registry
	aws ecr get-login-password --region $(REGION) | docker login --username AWS --password-stdin $(IMAGE_REGISTRY)

logout:	## Logout from remote image registry
	docker logout $(IMAGE_REGISTRY)

login-ecr-public: ## Login to ECR public
	aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin $(ECR_PUBLIC)

logout-ecr-public: ## Logout from ECR public
	docker logout $(ECR_PUBLIC)

push-server:	meta	## Push to remote image registry
	$(info    [PUSH_CONTAINER_IMAGE])
	docker push $(IMAGE_REGISTRY)/$(SERVER_IMAGE_REPO):$(VERSION)
	$(info	)

pull-server:	meta	## Pull from remote image registry
	$(info    [PULL_CONTAINER_IMAGE])
	docker pull $(IMAGE_REGISTRY)/$(SERVER_IMAGE_REPO):$(VERSION)
	$(info	)

push-init:	meta	## Push to remote image registry
	$(info    [PUSH_CONTAINER_IMAGE])
	docker push $(IMAGE_REGISTRY)/$(INIT_IMAGE_REPO):$(VERSION)
	$(info	)

pull-init:	meta	## Pull from remote image registry
	$(info    [PULL_CONTAINER_IMAGE])
	docker pull $(IMAGE_REGISTRY)/$(INIT_IMAGE_REPO):$(VERSION)
	$(info	)

##@ Local Development
compile-server:	clean	meta	## Compile server for local MacOS
	$(info   [COMPILE])
	go env -w GOPROXY=direct && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./cmd/bin/server/main ./cmd/server/main.go
	$(info	)

##@ Local Development
compile-init:	clean	meta	## Compile init for local MacOS
	$(info   [COMPILE])
	go env -w GOPROXY=direct && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./cmd/bin/init/main ./cmd/init/main.go
	$(info	)

clean:	## Remove compile binary
	-@rm cmd/bin/init/main
	-@rm cmd/bin/server/main

init:	## Initialize Go project
	-@rm go.mod
	-@rm go.sum
	go mod init $(GO_MOD_PATH)
	go mod tidy

check:	## Vet and Lint Go codebase
	-go vet main
	-golangci-lint run

test:	## Run tests
	go test $(GO_MOD_PATH) -test.v

run:	## Run local binary
	./main.bin -f server-config.yaml

##@ Metadata
meta:	## Provides metadata for other commands. Can be called as a target, but is mostly used by other targets as a dependency
	$(info    [METADATA])
	$(info    timestamp: [$(TIMESTAMP)])
	$(info    git commit: [$(GIT_COMMIT)])
	$(info    git URL: [$(GIT_URL)])
	$(info    Container image version: [$(VERSION)])
	$(info	)
