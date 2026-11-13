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

# Tests for the tag and push subcommands.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/testing.sh"
setup_tmp
make_fixture

base="$(git -C "$WORK" rev-parse HEAD)"

# A dirty working tree must be rejected before anything is created.
echo "// dirty" >> "${WORK}/kpt.go"
run_release tag v1.1.0
assert_eq 1 "$RC" "dirty tree is rejected"
assert_eq "" "$(git -C "$WORK" tag -l 'api/v1.1.0')" "no tag created for a dirty tree"
git -C "$WORK" checkout -q kpt.go

run_release tag v1.1.0
assert_eq 0 "$RC" "tag succeeds: ${OUT}"

api_commit="$(git -C "$WORK" rev-parse 'api/v1.1.0^{commit}')"
root_commit="$(git -C "$WORK" rev-parse 'v1.1.0^{commit}')"
cli_commit="$(git -C "$WORK" rev-parse 'cli/v1.1.0^{commit}')"
assert_eq "$base" "$api_commit" "api tag is on the unchanged commit"
assert_eq "$root_commit" "$(git -C "$WORK" rev-parse HEAD~1)" "root tag is on the first bump commit"
assert_eq "$cli_commit" "$(git -C "$WORK" rev-parse HEAD)" "cli tag is on HEAD"
assert_eq "$base" "$(git -C "$WORK" rev-parse HEAD~2)" "exactly two bump commits were added"

root_mod="$(git -C "$WORK" show v1.1.0:go.mod)"
assert_contains "$root_mod" "github.com/kptdev/kpt/api v1.1.0" "root go.mod requires the new api"
cli_mod="$(git -C "$WORK" show cli/v1.1.0:cli/go.mod)"
assert_contains "$cli_mod" "github.com/kptdev/kpt v1.1.0" "cli go.mod requires the new root"
assert_contains "$cli_mod" "github.com/kptdev/kpt/api v1.1.0" "cli go.mod requires the new api"
assert_not_contains "$cli_mod" "replace" "cli go.mod has no replace directive"
assert_contains "$(git -C "$WORK" show cli/v1.1.0:cli/go.sum)" "github.com/kptdev/kpt v1.1.0 h1:" "cli go.sum has the root hash"
assert_contains "$(git -C "$WORK" log -1 --format=%B)" "Signed-off-by:" "bump commits are signed off"

# Dry run pushes nothing.
DRY_RUN=true run_release push v1.1.0 main
assert_eq 0 "$RC" "dry run succeeds"
assert_eq "" "$(git -C "$ORIGIN" tag -l '*v1.1.0')" "dry run pushed no tags"
assert_eq "$base" "$(git -C "$ORIGIN" rev-parse main)" "dry run did not move the branch"

# A moved remote branch must make the atomic push fail without pushing any tag.
other="${TMP}/other"
git clone -q "$ORIGIN" "$other"
(cd "$other" && echo x > x.txt && git add x.txt && git commit -q -m other && git push -q origin main)
run_release push v1.1.0 main
assert_eq 1 "$RC" "push to a moved branch is rejected"
assert_eq "" "$(git -C "$ORIGIN" tag -l '*v1.1.0')" "no tag was pushed when the push was rejected"

# Once the local branch contains the remote commit again, the push goes through.
git -C "$WORK" pull -q --rebase origin main
run_release push v1.1.0 main
assert_eq 0 "$RC" "push after rebase succeeds: ${OUT}"
assert_eq "api/v1.1.0 cli/v1.1.0 v1.1.0" "$(git -C "$ORIGIN" tag -l '*v1.1.0' | sort | tr '\n' ' ' | sed 's/ $//')" "all three tags are pushed"
assert_eq "$(git -C "$WORK" rev-parse HEAD)" "$(git -C "$ORIGIN" rev-parse main)" "branch is pushed"

finish
