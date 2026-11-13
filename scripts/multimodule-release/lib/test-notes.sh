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

# Tests for the notes subcommand.
#
# Known history on top of v1.0.0, api/v1.0.0 and cli/v1.0.0 (PR numbers as in
# squash merge subjects):
#   #12 touches api/  <- api/v1.0.1-pre.1 is tagged here
#   #13 touches cli/
#   #15 touches api/
#   #14 touches the root
# Expected: api/ notes since api/v1.0.1-pre.1 have #15 only, cli/ notes since
# cli/v1.0.0 have #13 only.
# gh is stubbed: it returns notes listing all four PRs and records its arguments.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/testing.sh"
setup_tmp
make_fixture

(
  cd "$WORK" || exit 1
  echo a > api/a.txt && git add -A && git commit -q -m "feat api (#12)"
  git tag api/v1.0.1-pre.1
  echo c > cli/c.txt && git add -A && git commit -q -m "feat cli (#13)"
  echo b > api/b.txt && git add -A && git commit -q -m "fix api (#15)"
  echo r > r.txt && git add -A && git commit -q -m "other (#14)"
  git push -q origin main api/v1.0.1-pre.1
)

mkdir -p "${TMP}/bin"
cat > "${TMP}/bin/gh" <<'STUB'
#!/usr/bin/env bash
echo "$*" >> "${GH_STUB_LOG}"
cat <<'BODY'
## What's Changed
* feat api by @a in https://github.com/kptdev/kpt/pull/12
* feat cli by @a in https://github.com/kptdev/kpt/pull/13
* other by @a in https://github.com/kptdev/kpt/pull/14
* fix api by @a in https://github.com/kptdev/kpt/pull/15

**Full Changelog**: https://github.com/kptdev/kpt/compare/v1.0.0...v1.1.0
BODY
STUB
chmod +x "${TMP}/bin/gh"
export PATH="${TMP}/bin:${PATH}"
export GH_STUB_LOG="${TMP}/gh.log"

section() { # NOTES MODULE: the lines of the "Changes in MODULE/" section
  awk -v h="## Changes in \`$2/\`" '$0 == h {on=1; next} /^## /{on=0} on' <<< "$1"
}

check_notes() { # LABEL: verifies ${TMP}/notes.md and the gh calls
  local label="$1" notes calls
  notes="$(cat "${TMP}/notes.md")"
  calls="$(cat "$GH_STUB_LOG")"

  # Full notes: every PR, as returned for the previous root release.
  assert_contains "$notes" "pull/14" "${label}: full notes contain the root PR"
  assert_contains "$calls" "tag_name=v1.1.0 -f previous_tag_name=v1.0.0 -f target_commitish=main" \
    "${label}: full notes are requested since the previous root release"

  # Scoped notes: only PRs that touched the module since its previous tag.
  assert_contains "$calls" "tag_name=api/v1.1.0 -f previous_tag_name=api/v1.0.1-pre.1 -f target_commitish=main" \
    "${label}: api notes are requested since the highest earlier api tag"
  assert_contains "$calls" "tag_name=cli/v1.1.0 -f previous_tag_name=cli/v1.0.0 -f target_commitish=main" \
    "${label}: cli notes are requested since the previous cli tag"

  local api_notes cli_notes
  api_notes="$(section "$notes" api)"
  cli_notes="$(section "$notes" cli)"
  assert_contains "$api_notes" "pull/15" "${label}: api section has the api PR after the previous api tag"
  assert_not_contains "$api_notes" "pull/12" "${label}: api section has no PR from before the previous api tag"
  assert_not_contains "$api_notes" "pull/13" "${label}: api section has no cli PR"
  assert_not_contains "$api_notes" "pull/14" "${label}: api section has no root PR"
  assert_contains "$cli_notes" "pull/13" "${label}: cli section has the cli PR"
  assert_not_contains "$cli_notes" "pull/12" "${label}: cli section has no api PR"
  assert_not_contains "$cli_notes" "pull/15" "${label}: cli section has no api PR"
}

# Dry run: the new tags exist locally only (and the bump commits are not pushed).
run_release tag v1.1.0
assert_eq 0 "$RC" "tag succeeds: ${OUT}"
: > "$GH_STUB_LOG"
run_release notes v1.1.0 main "${TMP}/notes.md"
assert_eq 0 "$RC" "notes succeed before the push: ${OUT}"
check_notes "before push"
cp "${TMP}/notes.md" "${TMP}/notes-before-push.md"

# Real run: the same notes after the new tags were pushed.
run_release push v1.1.0 main
assert_eq 0 "$RC" "push succeeds: ${OUT}"
: > "$GH_STUB_LOG"
run_release notes v1.1.0 main "${TMP}/notes.md"
assert_eq 0 "$RC" "notes succeed after the push: ${OUT}"
check_notes "after push"
assert_eq "$(cat "${TMP}/notes-before-push.md")" "$(cat "${TMP}/notes.md")" \
  "notes are the same before and after pushing the tags"

finish
