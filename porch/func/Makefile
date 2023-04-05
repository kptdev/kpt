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

GCP_PROJECT_ID ?= $(shell gcloud config get-value project)
IMAGE_TAG  ?= latest
IMAGE_REPO ?= gcr.io/$(GCP_PROJECT_ID)
IMAGE_NAME ?= function-runner
WRAPPER_SERVER_IMAGE_NAME ?= wrapper-server
COMPILED_PROTO=evaluator/evaluator_grpc.pb.go evaluator/evaluator.pb.go

KPTDIR = $(abspath $(CURDIR)/../..)

all: $(COMPILED_PROTO)

$(COMPILED_PROTO): evaluator/evaluator.proto
	protoc \
	  -I /usr/local/include/google/protobuf \
	  -I ./evaluator \
	  --go_out=./evaluator --go_opt=paths=source_relative \
	  --go-grpc_out=./evaluator --go-grpc_opt=paths=source_relative \
	  ./evaluator/evaluator.proto 

.PHONY: build-image
build-image:
	docker buildx build --load --tag $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) -f ./Dockerfile "$(KPTDIR)"
	docker buildx build --load --tag $(IMAGE_REPO)/$(WRAPPER_SERVER_IMAGE_NAME):$(IMAGE_TAG) -f ./Dockerfile-wrapperserver "$(KPTDIR)"

.PHONY: push-image
push-image:
	docker buildx build --push --tag $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) -f ./Dockerfile "$(KPTDIR)"
	docker buildx build --push --tag $(IMAGE_REPO)/$(WRAPPER_SERVER_IMAGE_NAME):$(IMAGE_TAG) -f ./Dockerfile-wrapperserver "$(KPTDIR)"
