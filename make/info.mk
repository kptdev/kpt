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

GOLANG_VERSION    := 1.26.3
GOLANGCI_LINT_VERSION := 2.11.4

GORELEASER_CONFIG = release/tag/goreleaser.yaml
GORELEASER_IMAGE  := ghcr.io/goreleaser/goreleaser-cross:v$(GOLANG_VERSION)

YEAR_GEN          := $(shell date '+%Y')

GOBIN := $(shell go env GOPATH)/bin
KPT_VERSION := $(shell date '+development-%Y-%m-%dT%H:%M:%S')

export KPT_FN_WASM_RUNTIME ?= nodejs

LDFLAGS := -ldflags "-X github.com/kptdev/kpt/run.version=${KPT_VERSION}
ifeq ($(OS),Windows_NT)
	# Do nothing
else
    UNAME := $(shell uname -s)
    ifeq ($(UNAME),Linux)
        LDFLAGS += -extldflags '-z noexecstack'
    endif
endif
LDFLAGS += "

# T refers to an e2e test case matcher. This enables running e2e tests
# selectively.  For example,
# To invoke e2e tests related to fnconfig, run:
# make test-fn-render T=fnconfig
# make test-fn-eval T=fnconfig
# By default, make test-fn-render/test-fn-eval will run all tests.
T ?= ".*"
