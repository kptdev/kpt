#!/usr/bin/env bash
# Copyright 2021,2026 The kpt Authors
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

set -o errexit -o nounset -o pipefail -o posix -x

if ! command -v jq >/dev/null; then
  echo "jq must be installed. Follow https://stedolan.github.io/jq/download/ to install jq."
  exit 1
fi

if ! command -v yq >/dev/null; then
  echo "yq must be installed. Follow https://github.com/mikefarah/yq/releases to install yq."
  exit 1
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
BUILD_DIR="${REPO_ROOT}/.build"
mkdir -p ${BUILD_DIR}
cd "${REPO_ROOT}"

$GOBIN/swagger generate spec -m -w pkg/api/kptfile/v1 -o documentation/content/en/reference/schema/kptfile/kptfile.yaml
$GOBIN/swagger generate spec -m -w pkg/api/kptfile/v1 -o documentation/content/en/reference/schema/kptfile/kptfile.json

# Strip kubebuilder annotations from generated schema files.
jq -f ${REPO_ROOT}/scripts/strip-kubebuilder-annos.jq documentation/content/en/reference/schema/kptfile/kptfile.json > "${BUILD_DIR}/kptfile-schema.json"
yq -o json documentation/content/en/reference/schema/kptfile/kptfile.yaml \
  | jq -f ${REPO_ROOT}/scripts/strip-kubebuilder-annos.jq \
  | yq -P -I 4 -o yaml > "${BUILD_DIR}/kptfile-schema.yaml"

# We need to add schema header for schema to work in cloud-code.
# See https://github.com/kptdev/kpt/pull/2520/files/aac23473c121252ec6341fdb2bcce389ae6cb122#r717867089
jq -s '.[0] * .[1]' scripts/schema-header.json ${BUILD_DIR}/kptfile-schema.json > ${BUILD_DIR}/kptfile-schema-with-header.json

rm -f ${BUILD_DIR}/kptfile-schema.json
mv ${BUILD_DIR}/kptfile-schema-with-header.json documentation/content/en/reference/schema/kptfile/kptfile.json
mv ${BUILD_DIR}/kptfile-schema.yaml documentation/content/en/reference/schema/kptfile/kptfile.yaml
