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

.PHONY: generate license fix vet fmt lint test build tidy

GOBIN := $(shell go env GOPATH)/bin

build:
	go build -o $(GOPATH)/bin/kpt -v .

all: generate license fix vet fmt lint test build tidy docker

fix:
	go fix ./...

fmt:
	go fmt ./...

generate:
	(which $(GOBIN)/mdtogo || go get sigs.k8s.io/kustomize/cmd/mdtogo)
	rm -rf internal/docs/generated
	mkdir internal/docs/generated
	GOBIN=$(GOBIN) go generate ./...

tidy:
	go mod tidy

license:
	(which addlicense || go get github.com/google/addlicense)
	$(GOBIN)/addlicense  -y 2019 -l apache .

lint:
	(which golangci-lint || go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.18.0)
	$(GOBIN)/golangci-lint run ./...

test:
	go test -cover ./...

vet:
	go vet ./...

docker:
	docker build .