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

FROM gcr.io/kpt-fn/apply-setters:v0.2.0 as apply-setters
FROM gcr.io/kpt-fn/apply-setters:v0.1.1 as apply-setters-v1
FROM gcr.io/kpt-fn/ensure-name-substring:v0.1.1 as ensure-name-substring
FROM gcr.io/kpt-fn/search-replace:v0.2.0 as search-replace
FROM gcr.io/kpt-fn/set-annotations:v0.1.4 as set-annotations
FROM gcr.io/kpt-fn/set-image:v0.1.0 as set-image
FROM gcr.io/kpt-fn/set-labels:v0.1.5 as set-labels
FROM gcr.io/kpt-fn/set-namespace:v0.2.0 as set-namespace
FROM gcr.io/kpt-fn/set-namespace:v0.3.4 as set-namespace-v3
FROM gcr.io/kpt-fn/set-project-id:v0.2.0 as set-project-id
FROM gcr.io/kpt-fn/starlark:v0.3.0 as starlark
FROM gcr.io/kpt-fn/upsert-resource:v0.2.0 as upsert-resource

FROM golang:1.21.5-alpine3.18 as builder
WORKDIR /go/src/github.com/GoogleContainerTools/kpt

RUN go install github.com/grpc-ecosystem/grpc-health-probe@v0.4.11
RUN cp $GOPATH/bin/grpc-health-probe /grpc-health-probe

COPY go.mod go.sum ./
COPY porch/go.mod porch/go.sum porch/
COPY porch/api/go.mod porch/api/go.sum porch/api/

RUN cd porch ; go build -v google.golang.org/grpc k8s.io/klog/v2

COPY pkg pkg
COPY internal internal
COPY porch/func/ porch/func/
RUN cd porch/func ; go build -v -o /server ./server
COPY porch/func/config.yaml /config.yaml

FROM alpine:3.18

COPY --from=apply-setters          /usr/local/bin/function /functions/apply-setters
COPY --from=apply-setters-v1       /usr/local/bin/function /functions/apply-setters-v1
COPY --from=ensure-name-substring  /usr/local/bin/function /functions/ensure-name-substring
COPY --from=search-replace         /usr/local/bin/function /functions/search-replace
COPY --from=set-annotations        /usr/local/bin/function /functions/set-annotations
COPY --from=set-image              /usr/local/bin/function /functions/set-image
COPY --from=set-labels             /usr/local/bin/function /functions/set-labels
COPY --from=set-namespace          /usr/local/bin/function /functions/set-namespace
COPY --from=set-namespace-v3       /usr/local/bin/function /functions/set-namespace-v3
COPY --from=set-project-id         /usr/local/bin/function /functions/set-project-id
COPY --from=starlark               /usr/local/bin/star     /functions/starlark
COPY --from=upsert-resource        /usr/local/bin/function /functions/upsert-resource
COPY --from=builder                /server /grpc-health-probe /config.yaml /

EXPOSE 9445/tcp
ENTRYPOINT [ "/server", "--config=/config.yaml", "--functions=/functions" ]
