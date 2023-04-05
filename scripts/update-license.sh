#!/usr/bin/env bash
# Copyright 2019 The kpt Authors
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

set -o nounset
set -o errexit
set -o pipefail

# TODO: switch to google/addlicense once we have https://github.com/google/addlicense/pull/104
go run github.com/justinsb/addlicense@v1.0.1 \
  -c "The kpt Authors" -l apache \
  --ignore ".build/**" \
  --ignore "site/**" \
  --ignore "docs/**" \
  --ignore "**/.expected/results.yaml" \
  --ignore "**/testdata/**" \
  --ignore "**/generated/**" \
  --ignore "package-examples/cert-manager-basic/**" \
  --ignore "package-examples/ghost/**" \
  --ignore "package-examples/ingress-nginx/**" \
  --ignore "tools/licensescan/modules/**" \
  . 2>&1 | ( grep -v "skipping: " || true )
