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
# Tests for the assets and notes subcommands. Run: make test-scripts

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/testing.sh"
setup_tmp

# --- assets ---
mkdir -p "${TMP}/dist/linux-amd64_linux_amd64_v1" "${TMP}/dist/darwin-arm64_darwin_arm64_v8.0"
WORK="$TMP"
cd "$TMP" || exit 1
mkdir -p scripts
cp -r "$RELEASE_SCRIPTS_DIR" scripts/multimodule-release
for f in dist/kpt_linux_amd64-1.2.3.tar.gz dist/kpt_linux_amd64-1.2.3.deb dist/checksums.txt dist/metadata.json \
  dist/linux-amd64_linux_amd64_v1/kpt dist/darwin-arm64_darwin_arm64_v8.0/kpt; do
  echo "$f" > "$f"
done
cat > dist/artifacts.json <<'JSON'
[
  {"name":"kpt_linux_amd64-1.2.3.tar.gz","path":"dist/kpt_linux_amd64-1.2.3.tar.gz","type":"Archive"},
  {"name":"kpt_linux_amd64-1.2.3.deb","path":"dist/kpt_linux_amd64-1.2.3.deb","type":"Linux Package"},
  {"name":"checksums.txt","path":"dist/checksums.txt","type":"Checksum"},
  {"name":"metadata.json","path":"dist/metadata.json","type":"Metadata"},
  {"name":"kpt","path":"dist/linux-amd64_linux_amd64_v1/kpt","type":"Binary","goos":"linux","goarch":"amd64"},
  {"name":"kpt","path":"dist/linux-amd64_linux_amd64_v1/kpt","type":"Binary","goos":"linux","goarch":"amd64"},
  {"name":"kpt","path":"dist/darwin-arm64_darwin_arm64_v8.0/kpt","type":"Binary","goos":"darwin","goarch":"arm64"}
]
JSON
git init -q -b main . >/dev/null
run_release assets dist out
assert_eq 0 "$RC" "assets succeeds: ${OUT}"
assert_eq "checksums.txt kpt_darwin_arm64 kpt_linux_amd64 kpt_linux_amd64-1.2.3.deb kpt_linux_amd64-1.2.3.tar.gz" \
  "$(ls out | tr '\n' ' ' | sed 's/ $//')" "release assets are staged with their upload names"
assert_eq "dist/linux-amd64_linux_amd64_v1/kpt" "$(cat out/kpt_linux_amd64)" "plain binary content is copied"

run_release assets nonexistent out2
assert_eq 1 "$RC" "missing artifacts.json is rejected"

# --- notes ---
cd "$SCRIPT_DIR" || exit 1
rm -rf "$TMP/dist" "$TMP/out" "$TMP/scripts" "$TMP/.git"
make_fixture
(
  cd "$WORK" || exit 1
  echo a > api/a.txt && git add -A && git commit -q -m "feat api (#12)"
  echo c > cli/c.txt && git add -A && git commit -q -m "feat cli (#13)"
  echo r > r.txt && git add -A && git commit -q -m "other (#14)"
  git push -q origin main
)

# gh is stubbed: the generate-notes API call returns the same body every time.
mkdir -p "${TMP}/bin"
cat > "${TMP}/bin/gh" <<'STUB'
#!/usr/bin/env bash
cat <<'BODY'
## What's Changed
* feat api by @a in https://github.com/kptdev/kpt/pull/12
* feat cli by @a in https://github.com/kptdev/kpt/pull/13
* other by @a in https://github.com/kptdev/kpt/pull/14

**Full Changelog**: https://github.com/kptdev/kpt/compare/v1.0.0...v1.1.0
BODY
STUB
chmod +x "${TMP}/bin/gh"
export PATH="${TMP}/bin:${PATH}"

run_release tag v1.1.0
assert_eq 0 "$RC" "tag succeeds: ${OUT}"
run_release push v1.1.0 main
assert_eq 0 "$RC" "push succeeds: ${OUT}"

run_release notes v1.1.0 "${TMP}/notes.md"
assert_eq 0 "$RC" "notes succeeds: ${OUT}"
notes="$(cat "${TMP}/notes.md")"
section() { awk -v h="## Changes in \`$1/\`" '$0 == h {on=1; next} /^## /{on=0} on' <<< "$notes"; }
api_notes="$(section api)"
cli_notes="$(section cli)"
assert_contains "$notes" "pull/14" "full notes contain every PR"
assert_contains "$api_notes" "pull/12" "api section has the api PR"
assert_not_contains "$api_notes" "pull/13" "api section has no cli PR"
assert_not_contains "$api_notes" "pull/14" "api section has no root PR"
assert_contains "$cli_notes" "pull/13" "cli section has the cli PR"
assert_not_contains "$cli_notes" "pull/12" "cli section has no api PR"

finish
