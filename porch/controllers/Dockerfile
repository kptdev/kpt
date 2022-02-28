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

WORKDIR /workspace
COPY go.mod go.sum ./
COPY porch/api/go.mod porch/api/go.sum porch/api/
COPY porch/controllers/go.mod porch/controllers/go.sum porch/controllers/
COPY porch/controllers/remoterootsync/go.mod porch/controllers/remoterootsync/go.sum porch/controllers/remoterootsync/
COPY porch/repository/go.mod porch/repository/go.sum porch/repository/

WORKDIR /workspace/porch/controllers/remoterootsync/
RUN go mod download
# Prebuild some libraries to warm the cache
RUN CGO_ENABLED=0 go build -v \
  k8s.io/klog/v2 \
  k8s.io/klog/v2/klogr \
  k8s.io/client-go/plugin/pkg/client/auth \
  sigs.k8s.io/controller-runtime \
  sigs.k8s.io/controller-runtime/pkg/client \
  sigs.k8s.io/controller-runtime/pkg/controller/controllerutil \
  k8s.io/client-go/kubernetes \
  go.opentelemetry.io/otel \
  cloud.google.com/go/container/apiv1 \
  github.com/google/go-containerregistry/pkg/gcrane \
  github.com/google/go-containerregistry/pkg/v1 \
  github.com/google/go-containerregistry/pkg/v1/cache \
  k8s.io/client-go/discovery/cached

WORKDIR /workspace
COPY internal/ internal/
COPY pkg/ pkg/
COPY porch/api/ porch/api/
COPY porch/controllers/ porch/controllers/
COPY porch/repository/ porch/repository/

WORKDIR /workspace/porch/controllers/remoterootsync/
RUN CGO_ENABLED=0 go build -o /porch-controllers -v .

FROM gcr.io/distroless/static
WORKDIR /data
COPY --from=builder /porch-controllers /porch-controllers

ENTRYPOINT ["/porch-controllers"]
