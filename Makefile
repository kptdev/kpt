# Copyright 2019 Google LLC
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

.PHONY: docs license fix vet fmt lint test build tidy

GOBIN := $(shell go env GOPATH)/bin

build:
	go build -o $(GOBIN)/kpt -v .

all: license fix vet fmt lint test build buildall tidy

buildall:
	GOOS=windows go build -o /dev/null
	GOOS=linux go build -o /dev/null
	GOOS=darwin go build -o /dev/null

update-deps-to-head:
	go get sigs.k8s.io/cli-utils@master
	go get sigs.k8s.io/kustomize/cmd/config@master
	go get sigs.k8s.io/kustomize/kyaml@master

fix:
	go fix ./...

fmt:
	go fmt ./...

generate:
	go install ./mdtogo
	rm -rf internal/docs/generated
	mkdir internal/docs/generated
	rm -rf internal/guides/generated
	mkdir internal/guides/generated
	GOBIN=$(GOBIN) go generate ./...
	which addlicense || go get github.com/google/addlicense
	$(GOBIN)/addlicense -y 2019 -l apache internal/docs/generated
	$(GOBIN)/addlicense -y 2019 -l apache internal/guides/generated
	go fmt ./internal/docs/generated/...
	go fmt ./internal/guides/generated/...

tidy:
	go mod tidy

license:
	GOBIN=$(GOBIN) scripts/update-license.sh

lint:
	(which golangci-lint || go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.31.0)
	$(GOBIN)/golangci-lint run ./...

# TODO: enable this as part of `all` target when it works for go-errors
# https://github.com/google/go-licenses/issues/15
license-check:
	(which go-licensesscs || go get https://github.com/google/go-licenses)
	$(GOBIN)/go-licenses check github.com/GoogleContainerTools/kpt

test:
	go test -cover ./...

vet:
	go vet ./...

docker:
	docker build .

functions-examples-docker:
	docker build . -f functions/examples/Dockerfile -t gcr.io/kpt-dev/example-functions:v0.1.0
	docker push gcr.io/kpt-dev/example-functions:v0.1.0

lintdocs:
	(cd site && npm run lint-fix)

gencatalog:
	rm site/content/en/guides/consumer/function/catalog/*/_index.md
	(cd site/content/en/guides/consumer/function/catalog/catalog && npm run gen-docs)

servedocs:
	(cd site && go run github.com/gohugoio/hugo server)

verify-guides:
	./scripts/verifyGuides.sh
