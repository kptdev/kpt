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
# Unit tests for lib/semver.sh. Run: scripts/multimodule-release/test-semver.sh

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"
source "${SCRIPT_DIR}/lib/semver.sh"

failures=0
expect_cmp() {
  local got
  got="$(semver_cmp "$1" "$2")"
  if [[ "$got" != "$3" ]]; then
    echo "FAIL: semver_cmp $1 $2 = $got, want $3"
    failures=$((failures + 1))
  fi
}
expect_valid() {
  if is_semver "$1"; then got=valid; else got=invalid; fi
  if [[ "$got" != "$2" ]]; then
    echo "FAIL: is_semver $1 = $got, want $2"
    failures=$((failures + 1))
  fi
}

expect_cmp v1.0.0 v1.0.0 0
expect_cmp v1.0.1 v1.0.0 1
expect_cmp v1.10.0 v1.9.0 1
expect_cmp v1.0.0 v1.0.0-pre.1 1
expect_cmp v1.0.0-pre.1 v1.0.0 -1
expect_cmp v1.0.2-pre.4 v1.0.2-pre.10 -1
expect_cmp v1.0.0-alpha v1.0.0-alpha.1 -1
expect_cmp v1.0.0-alpha.1 v1.0.0-alpha.beta -1
expect_cmp v1.0.0-beta.2 v1.0.0-beta.11 -1
expect_cmp v1.0.0-beta v1.0.0-alpha 1
expect_cmp 1.1.1 v1.1.1-pre.2 1
expect_cmp v1.0.0-beta.7 v1.0.0-beta.10 -1

expect_valid v1.2.3 valid
expect_valid 1.2.3 valid
expect_valid v1.2.3-pre.1 valid
expect_valid v10.20.30-rc.1 valid
expect_valid v1.2 invalid
expect_valid v01.2.3 invalid
expect_valid v1.2.3+meta invalid
expect_valid v1.2.3-01 invalid
expect_valid latest invalid
expect_valid "v1.2.3 " invalid

tags=$'v1.0.2-pre.4\nv1.0.2-pre.10\nv1.0.1\napi/v1.1.1-pre.2\napi/v1.1.0\ncli/v1.0.2-pre.4\nporch/v9.0.0'
[[ "$(latest_semver_tag "" <<< "$tags")" == v1.0.2-pre.10 ]] || { echo "FAIL: latest root"; failures=$((failures + 1)); }
[[ "$(latest_semver_tag "api/" <<< "$tags")" == api/v1.1.1-pre.2 ]] || { echo "FAIL: latest api"; failures=$((failures + 1)); }
[[ "$(latest_semver_tag "cli/" <<< "$tags")" == cli/v1.0.2-pre.4 ]] || { echo "FAIL: latest cli"; failures=$((failures + 1)); }

if ((failures > 0)); then
  echo "$failures failure(s)"
  exit 1
fi
echo "All semver tests passed"
