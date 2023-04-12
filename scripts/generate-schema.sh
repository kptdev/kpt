#!/usr/bin/env bash
# Copyright 2021 The kpt Authors
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

set -o errexit -o nounset -o pipefail -o posix

if ! command -v jq >/dev/null; then
  echo "jq must be installed. Follow https://stedolan.github.io/jq/download/ to install jq."
  exit 1
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "${REPO_ROOT}"

make install-swagger

$GOBIN/swagger generate spec -m -w pkg/api/kptfile/v1 -o site/reference/schema/kptfile/kptfile.yaml
$GOBIN/swagger generate spec -m -w pkg/api/kptfile/v1 -o site/reference/schema/kptfile/kptfile.json

# We need to add schema header for schema to work in cloud-code.
# See https://github.com/GoogleContainerTools/kpt/pull/2520/files/aac23473c121252ec6341fdb2bcce389ae6cb122#r717867089
jq -s '.[0] * .[1]' scripts/schema-header.json site/reference/schema/kptfile/kptfile.json > /tmp/kptfile-schema.json
mv /tmp/kptfile-schema.json site/reference/schema/kptfile/kptfile.json