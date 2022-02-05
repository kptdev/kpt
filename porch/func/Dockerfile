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

FROM gcr.io/kpt-fn/apply-setters:v0.2.0 as apply-setters
FROM gcr.io/kpt-fn/ensure-name-substring:v0.1.1 as ensure-name-substring
FROM gcr.io/kpt-fn/search-replace:v0.2.0 as search-replace
FROM gcr.io/kpt-fn/set-annotations:v0.1.4 as set-annotations
FROM gcr.io/kpt-fn/set-image:unstable as set-image
FROM gcr.io/kpt-fn/set-labels:v0.1.5 as set-labels
FROM gcr.io/kpt-fn/set-namespace:v0.2.0 as set-namespace
FROM gcr.io/kpt-fn/set-project-id:v0.2.0 as set-project-id
FROM gcr.io/kpt-fn/upsert-resource:v0.2.0 as upsert-resource

FROM golang:1.17.6-alpine3.15 as builder
WORKDIR /build
COPY . ./func
RUN cd ./func ; go mod download -x
RUN cd ./func ; go build -v -o /server ./server

FROM alpine:3.15

COPY --from=apply-setters          /usr/local/bin/function /functions/apply-setters
COPY --from=ensure-name-substring  /usr/local/bin/function /functions/ensure-name-substring
COPY --from=search-replace         /usr/local/bin/function /functions/search-replace
COPY --from=set-annotations        /usr/local/bin/function /functions/set-annotations
COPY --from=set-image              /usr/local/bin/function /functions/set-image
COPY --from=set-labels             /usr/local/bin/function /functions/set-labels
COPY --from=set-namespace          /usr/local/bin/function /functions/set-namespace
COPY --from=set-project-id         /usr/local/bin/function /functions/set-project-id
COPY --from=upsert-resource        /usr/local/bin/function /functions/upsert-resource
COPY --from=builder                /server /build/func/config.yaml /

EXPOSE 9445/tcp
ENTRYPOINT [ "/server", "--config=/config.yaml", "--functions=/functions" ]
