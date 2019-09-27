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

all: generate license fix vet fmt lint test build tidy

build:
	(cd cmd/ && go build -o kpt -v .)

fix:
	(cd cmd/ && go fix ./...)
	(cd lib/ &&  go fix ./... )

fmt:
	(cd cmd/ && go fmt ./...)
	(cd lib/ &&  go fmt ./... )

generate:
	(cd cmd/ && go generate ./...)
	(cd lib// && go generate ./...)

tidy:
	(cd cmd/ && go mod tidy)
	(cd lib/ && go mod tidy)

license:
	(which addlicense || go get github.com/google/addlicense)
	(cd cmd/ && addlicense  -y 2019 -l apache .)
	(cd lib/ &&  addlicense  -y 2019 -l apache .)

lint:
	(which golangci-lint || go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.18.0)
	(cd cmd/ && golangci-lint run ./...)
	(cd lib/ && golangci-lint run ./...)

test:
	(cd cmd/ && go test -cover ./...)
	(cd lib/ && go test -cover ./...)

vet:
	(cd cmd/ && go vet ./...)
	(cd lib/ &&  go vet ./... )
