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

GOPATH := $(shell go env GOPATH)

build:
	(cd kpt/ && go build -o $(GOPATH)/bin/kpt -v .)

all: generate license fix vet fmt lint test build tidy

fix:
	(cd kpt/ && go fix ./...)
	(cd lib/ &&  go fix ./... )

fmt:
	(cd kpt/ && go fmt ./...)
	(cd lib/ &&  go fmt ./... )

generate:
	rm -rf kpt/cmdtutorials/generated
	rm -rf kpt/generated
	(cd kpt/ && go generate ./...)
	(cd lib/ && go generate ./...)

tidy:
	(cd kpt/ && go mod tidy)
	(cd lib/ && go mod tidy)

license:
	(which addlicense || go get github.com/google/addlicense)
	$(GOPATH)/bin/addlicense  -y 2019 -l apache .

lint:
	(which golangci-lint || go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.18.0)
	(cd kpt/ && $(GOPATH)/bin/golangci-lint run ./...)
	(cd lib/ && $(GOPATH)/bin/golangci-lint run ./...)

test:
	(cd kpt/ && go test -cover ./...)
	(cd lib/ && go test -cover ./...)

vet:
	(cd kpt/ && go vet ./...)
	(cd lib/ &&  go vet ./... )
