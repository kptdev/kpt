# Copyright 2022 Google LLC
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


FROM golang:1.17-bullseye as builder

WORKDIR /go/src/github.com/GoogleContainerTools/kpt

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer

COPY go.mod go.sum ./
COPY porch/apiserver/go.mod porch/apiserver/go.sum porch/apiserver/
COPY porch/engine/go.mod porch/engine/go.mod porch/engine/
COPY porch/repository/go.mod porch/repository/go.sum porch/repository/
COPY porch/func/go.mod porch/func/go.sum porch/func/

RUN echo "Downloading root modules ..." \
 && go mod download
RUN echo "Downloading apiserver modules ..." \
 && cd porch/apiserver && go mod download
RUN echo "Downloading engine modules ..." \
 && cd porch/engine && go mod download
RUN echo "Downloading repository modules ..." \
 && cd porch/repository && go mod download
RUN echo "Downloading func modules ..." \
 && cd porch/func && go mod download

# Prebuild some library dependencies to warm the cache
RUN cd porch/apiserver; go build -v \
  google.golang.org/grpc \
  k8s.io/apiserver/pkg/server \
  k8s.io/component-base/cli \
  k8s.io/klog/v2 \
  github.com/google/go-containerregistry/pkg/gcrane \
  k8s.io/client-go/kubernetes/scheme \
  github.com/go-git/go-git/v5

COPY internal internal
COPY pkg pkg
COPY porch/api porch/api
COPY porch/apiserver porch/apiserver
COPY porch/controllers porch/controllers
COPY porch/engine porch/engine
COPY porch/repository porch/repository
COPY porch/func porch/func

RUN cd porch/apiserver; go build -v -o /porch ./cmd/porch

FROM debian:bullseye
RUN apt update && apt install -y ca-certificates && rm -rf /var/lib/apt && rm -rf /var/cache/apt
COPY --from=builder /porch /porch
ENTRYPOINT ["/porch"]
