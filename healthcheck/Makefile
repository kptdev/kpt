# Copyright 2023 The kpt Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Directories.
TOOLBIN := $(shell pwd)/bin
CONTROLLER_GEN = $(TOOLBIN)/controller-gen

.DEFAULT_GOAL := all
.PHONY: all
all: manifests generate vet fmt fix misspell tidy

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

##@ Development

manifests: $(TOOLBIN)/controller-gen ## Generate CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd paths="./..."

generate: $(TOOLBIN)/controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object paths="./..."


## --------------------------------------
## Code maintenance
## --------------------------------------

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: fix
fix:
	go fix ./api/...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: lint
lint: $(TOOLBIN)/golangci-lint
	$(TOOLBIN)/golangci-lint run ./...

.PHONY: misspell
misspell: $(TOOLBIN)/misspell
	$(TOOLBIN)/misspell ./**

.PHONY: misspell-fix
misspell-fix: $(TOOLBIN)/misspell
	$(TOOLBIN)/misspell -w ./**

.PHONY: fix-licenses
fix-licenses:
	../scripts/update-license.sh

## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(TOOLBIN)/controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(TOOLBIN)/controller-gen,sigs.k8s.io/controller-tools/cmd/controller-gen@v0.11.3)

$(TOOLBIN)/golangci-lint:
	$(call go-get-tool,$(TOOLBIN)/golangci-lint,github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.0)

$(TOOLBIN)/misspell:
	$(call go-get-tool,$(TOOLBIN)/misspell,github.com/client9/misspell/cmd/misspell@v0.3.4)	

.PHONY: install-tools
install-tools: \
	$(TOOLBIN)/controller-gen \
	$(TOOLBIN)/golangci-lint \
	$(TOOLBIN)/misspell

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

