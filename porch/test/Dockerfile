# Copyright 2022 The kpt Authors
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


FROM golang:1.21.5-bookworm as builder

WORKDIR /go/src/github.com/GoogleContainerTools/kpt

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer

COPY go.mod go.sum ./
COPY porch/go.mod porch/go.sum porch/
COPY porch/api/go.mod porch/api/go.sum porch/api/

RUN echo "Downloading root modules ..." \
 && go mod download
RUN echo "Downloading porch modules ..." \
 && cd porch && go mod download
RUN echo "Downloading api modules ..." \
 && cd porch/api && go mod download

# Prebuild some library dependencies to warm the cache
RUN cd porch/; go build -v \
  k8s.io/klog/v2 \
  github.com/go-git/go-git/v5

COPY internal internal
COPY pkg pkg
COPY porch/api porch/api
COPY porch/controllers porch/controllers
COPY porch/pkg porch/pkg
COPY porch/test porch/test

RUN cd porch/test; go build -v -o /git-server ./git

FROM debian:bookworm-slim
RUN apt update && apt install -y ca-certificates && rm -rf /var/lib/apt && rm -rf /var/cache/apt
COPY --from=builder /git-server /git-server
ENTRYPOINT ["/git-server", "--port=8080"]
