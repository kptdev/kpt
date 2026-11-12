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
# Tests for the validate subcommand. Run: make test-scripts

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/testing.sh"
setup_tmp
make_fixture

run_release validate 1.1.0 main
assert_eq 0 "$RC" "newer version is accepted"
assert_contains "$OUT" "v1.1.0" "version is printed with v prefix"

run_release validate v1.1.0-pre.1 main
assert_eq 0 "$RC" "prerelease newer than the last release is accepted"

(cd "$WORK" && git tag v2.0.0-pre.1 && git push -q origin v2.0.0-pre.1)

run_release validate v1.9.0 main
assert_eq 1 "$RC" "version older than a prerelease tag of a higher version is rejected"

run_release validate v2.0.0 main
assert_eq 0 "$RC" "release is newer than its prerelease tag"

run_release validate v2.0.0-pre.2 main
assert_eq 0 "$RC" "newer prerelease is accepted"

run_release validate v2.0.0-pre.1 main
assert_eq 1 "$RC" "existing tag is rejected"
assert_contains "$OUT" "already exists" "existing tag message"

run_release validate v1.0.5 main
assert_eq 1 "$RC" "older version is rejected"
assert_contains "$OUT" "not newer" "older version message"

run_release validate latest main
assert_eq 1 "$RC" "non-semver is rejected"

run_release validate v2.1.0+build main
assert_eq 1 "$RC" "build metadata is rejected"

run_release validate v2.1.0 nope
assert_eq 1 "$RC" "missing branch is rejected"
assert_contains "$OUT" "not found" "missing branch message"

# Only one module tag being ahead must also block the version.
(cd "$WORK" && git tag api/v3.0.0 && git push -q origin api/v3.0.0)
run_release validate v2.5.0 main
assert_eq 1 "$RC" "version older than an api tag is rejected"
assert_contains "$OUT" "api/v3.0.0" "blocking tag is named"

finish
