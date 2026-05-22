# Copyright 2026 The kpt Authors
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

SHELL := bash
.SHELLFLAGS := -exc

include ../make/info.mk

.PHONY: all
all: fix vet fmt lint tidy

.PHONY: fix
fix:
	go fix ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: generate
generate:
	GOBIN=$(GOBIN) YEAR_GEN=$(YEAR_GEN) go generate ./...

.PHONY: tidy
tidy:
	go mod tidy

# if the local version of golangci-lint matches the one we want, we can use that without overhead
.PHONY: lint
lint:
	@if [[ `command -v golangci-lint` && `golangci-lint version --short` == $(GOLANGCI_LINT_VERSION) ]]; then \
		golangci-lint run -v ./...; \
	else \
		go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION) run -v ./...; \
	fi

.PHONY: vet
vet:
	go vet ./...
