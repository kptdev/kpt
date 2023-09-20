#!/usr/bin/env bash

# Copyright 2023 The kpt Authors
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

set -o errexit
set -o pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}"/tools/generate-static-site

go build -o "${REPO_ROOT}/bin/generate-static-site" .

cd "${REPO_ROOT}"
rm -rf websites/kpt.dev
"${REPO_ROOT}/bin/generate-static-site"
cp -r site/static/ websites/kpt.dev/static/