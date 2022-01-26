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

FROM alpine:3.13
RUN apk update && apk upgrade && \
    apk add --no-cache git less mandoc diffutils bash openssh docker-cli && \
    rm -rf /var/lib/apt/lists/* && \
    rm /var/cache/apk/*
# This is set up for the Dockerfile to be used by goreleaser: https://goreleaser.com/customization/docker/
COPY kpt /usr/local/bin/kpt
ENTRYPOINT ["kpt"]
