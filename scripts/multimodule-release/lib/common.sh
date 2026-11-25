#!/usr/bin/env bash
# Copyright 2026 The kpt Authors
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

# Common helpers. Sourced by release.sh, do not execute directly.

# Modules released together: directory of the go.mod and the tag prefix.
# The root module is tagged without a prefix.
# shellcheck disable=SC2034 # used by the scripts sourcing this file
MODULES_API_DIR="api"
MODULES_CLI_DIR="cli"
API_MODULE="github.com/kptdev/kpt/api"
ROOT_MODULE="github.com/kptdev/kpt"

log() { echo "[INFO] $*" >&2; }
fatal() { echo "[ERR] $*" >&2; exit 1; }

# Lists tags of the remote (names only) so that unfetched tags are also seen.
remote_tags() {
  git ls-remote --tags --refs origin | sed 's#.*refs/tags/##'
}
