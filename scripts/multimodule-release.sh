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

set -eu

function fatal() {
  echo "$1"
  exit 1
}

function validate_semver() {
  if [[ ! ( "$1" =~ v[0-9]\.[0-9]\.[0-9](-pre\.[0-9])? ) ]]; then
    fatal "$1 is not a valid semantic version"
  fi
}

function check_branch_exists() {
  if ! git ls-remote --exit-code --heads origin "$1" 1>/dev/null; then
    fatal "Branch '$1' not found in remote 'origin'"
  fi
}

VERSION=${VERSION:-${1:?'VERSION must be set'}}
VERSION="v${VERSION#v}" # ensure v prefix
BRANCH=${BRANCH:-${2:-'main'}}

echo "Using branch ${BRANCH}"
echo "Using version ${VERSION}"

validate_semver "$VERSION"
check_branch_exists "$BRANCH"
