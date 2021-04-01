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

FROM golang:1.15-stretch AS build
ENV CGO_ENABLED=0
ENV GO111MODULE=on

ARG KPT_VERSION=unknown

WORKDIR /go/src/
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -v -o /usr/local/bin/kpt -ldflags="-s -w -X github.com/GoogleContainerTools/kpt/run.version=$KPT_VERSION" ./

FROM gcr.io/google.com/cloudsdktool/cloud-sdk:332.0.0-alpine
RUN apk update && apk upgrade && \
    apk add --no-cache git less diffutils bash openssh docker-cli && \
    rm -rf /var/lib/apt/lists/* && \
    rm /var/cache/apk/*
COPY --from=build /usr/local/bin/kpt /usr/local/bin/kpt
ENTRYPOINT ["kpt"]
