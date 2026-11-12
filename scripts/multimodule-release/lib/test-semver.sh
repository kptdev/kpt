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

# Unit tests for semver.sh. Run: make test-scripts

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/testing.sh"
source "${SCRIPT_DIR}/common.sh"
source "${SCRIPT_DIR}/semver.sh"

expect_cmp() { # A B WANT
  assert_eq "$3" "$(semver_cmp "$1" "$2")" "semver_cmp $1 $2"
}
expect_valid() { # VERSION valid|invalid
  local got=invalid
  if is_semver "$1"; then got=valid; fi
  assert_eq "$2" "$got" "is_semver '$1'"
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
assert_eq v1.0.2-pre.10 "$(latest_semver_tag "" <<< "$tags")" "latest root tag"
assert_eq api/v1.1.1-pre.2 "$(latest_semver_tag "api/" <<< "$tags")" "latest api tag"
assert_eq cli/v1.0.2-pre.4 "$(latest_semver_tag "cli/" <<< "$tags")" "latest cli tag"
assert_eq "" "$(latest_semver_tag "other/" <<< "$tags")" "no tag for an unknown prefix"

finish
