# Copyright 2019 The kpt Authors
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

GOLANG_VERSION         := 1.21.5
GORELEASER_CONFIG      = release/tag/goreleaser.yaml
GORELEASER_IMAGE       := ghcr.io/goreleaser/goreleaser-cross:v$(GOLANG_VERSION)

.PHONY: docs license fix vet fmt lint test build tidy release release-ci

GOBIN := $(shell go env GOPATH)/bin
GIT_COMMIT := $(shell git rev-parse --short HEAD)

LDFLAGS := -ldflags "-X github.com/GoogleContainerTools/kpt/run.version=${GIT_COMMIT}
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

all: generate license fix vet fmt lint license-check test build tidy

build:
	go build ${LDFLAGS} -o $(GOBIN)/kpt -v .

update-deps-to-head:
	go get sigs.k8s.io/cli-utils@master
	go get sigs.k8s.io/kustomize/kyaml@master

.PHONY: install-mdrip
install-mdrip:
	go install github.com/monopole/mdrip@v1.0.2

.PHONY: install-kind
install-kind:
	go install sigs.k8s.io/kind@v0.13.0

.PHONY: install-golangci-lint
install-golangci-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.2

.PHONY: install-go-licenses
install-go-licenses:
	go install github.com/google/go-licenses@v1.2.0

.PHONY: install-swagger
install-swagger:
	go install github.com/go-swagger/go-swagger/cmd/swagger@v0.27.0

.PHONY: install-mdtogo
install-mdtogo:
	go install ./mdtogo

fix:
	go fix ./...

fmt:
	go fmt ./...

schema:
	GOBIN=$(GOBIN) scripts/generate-schema.sh

generate: install-mdtogo
	rm -rf internal/docs/generated
	mkdir internal/docs/generated
	GOBIN=$(GOBIN) go generate ./...
	go fmt ./internal/docs/generated/...

tidy:
	go mod tidy

license:
	scripts/update-license.sh

lint: install-golangci-lint
	$(GOBIN)/golangci-lint run ./...

license-check: install-go-licenses
	$(GOBIN)/go-licenses check github.com/GoogleContainerTools/kpt

test:
	go test -cover ${LDFLAGS} ./...

# This target is used to run Go tests that require docker runtime.
# Some tests, like pipeline tests, need to have docker available to run.
# KPT_FN_RUNTIME can be set to select the desired function runtime.
# If unspecified, the default function runtime will be used.
test-docker: build
	PATH=$(GOBIN):$(PATH) go test -cover --tags=docker ./...

# KPT_E2E_UPDATE_EXPECTED=true (if expected output to be updated)
# target to run e2e tests for "kpt fn render" command
# KPT_FN_RUNTIME can be set to select the desired function runtime.
# If unspecified, the default function runtime will be used.
test-fn-render: build
	PATH=$(GOBIN):$(PATH) go test -v --tags=docker --run=TestFnRender/testdata/fn-render/$(T) ./e2e/

# target to run e2e tests for "kpt fn eval" command
# KPT_FN_RUNTIME can be set to select the desired function runtime.
# If unspecified, the default function runtime will be used.
test-fn-eval: build
	PATH=$(GOBIN):$(PATH) go test -v --tags=docker --run=TestFnEval/testdata/fn-eval/$(T)  ./e2e/

# target to run e2e tests for "kpt live apply" command
test-live-apply: build
	PATH=$(GOBIN):$(PATH) go test -v -timeout=20m --tags=kind -p 2 --run=TestLiveApply/testdata/live-apply/$(T)  ./e2e/

# target to run e2e tests for "kpt live plan" command
test-live-plan: build
	PATH=$(GOBIN):$(PATH) go test -v -timeout=20m --tags=kind -p 2 --run=TestLivePlan/testdata/live-plan/$(T)  ./e2e/

test-porch: build
	PATH=$(GOBIN):$(PATH) go test -v --count=1 --tags=porch ./e2e/

vet:
	go vet ./...

docker:
	docker build .

lintdocs:
	(cd site && npm run lint-fix)

site-generate:
	go run ./scripts/generate_site_sidebar > site/sidebar.md
	(cd site && find . -iname "00.md" -execdir ln -sf {} README.md \; && sed -i.bak s/00.md//g sidebar.md && rm sidebar.md.bak)

site-map:
	make site-run-server
	./scripts/generate-sitemap.sh

site-run-server:
	make site-generate
	./scripts/run-site.sh

site-check:
	make site-run-server
	./scripts/check-site.sh

site-verify-examples: install-mdrip install-kind
	./scripts/verifyExamples.sh

release-dry-run:
	@docker run \
		--rm \
		--privileged \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/github.com/GoogleContainerTools/kpt \
		-w /go/src/github.com/GoogleContainerTools/kpt \
		$(GORELEASER_IMAGE) \
		-f "$(GORELEASER_CONFIG)" \
		--skip-validate --skip-publish

release:
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	docker run \
		--rm \
		--privileged \
		--env-file .release-env \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/github.com/GoogleContainerTools/kpt \
		-w /go/src/github.com/GoogleContainerTools/kpt \
		$(GORELEASER_IMAGE) \
		-f "$(GORELEASER_CONFIG)" release \
		--skip-validate

release-ci:
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	docker run \
		--rm \
		--privileged \
		--env-file .release-env \
		-v ${HOME}/.docker/config.json:/root/.docker/config.json \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/github.com/GoogleContainerTools/kpt \
		-w /go/src/github.com/GoogleContainerTools/kpt \
		$(GORELEASER_IMAGE) \
		-f "$(GORELEASER_CONFIG)" release \
		--skip-validate

.PHONY: vulncheck
vulncheck: build
	# Scan the source
	GOFLAGS= go run golang.org/x/vuln/cmd/govulncheck@latest ./...
