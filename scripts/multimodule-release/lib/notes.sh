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
# Release notes. Sourced by release.sh, do not execute directly.
# Requires gh (authenticated via GH_TOKEN) and tags that exist on the remote.

# --- Subcommand: notes ---
# The notes consist of the repository wide GitHub generated notes, followed by
# the changes scoped to the api/ and cli/ modules. The previous release of
# each is the highest earlier tag with the matching prefix.
cmd_notes() {
  local version="v${1#v}" outfile="$2"
  local repo="${GITHUB_REPOSITORY:-kptdev/kpt}"
  local tags
  tags="$(remote_tags | grep -Fvx -e "$version" -e "api/${version}" -e "cli/${version}")"

  local prev_root
  prev_root="$(latest_semver_tag "" <<< "$tags")"

  {
    if [[ -n "$prev_root" ]]; then
      gh api "repos/${repo}/releases/generate-notes" \
        -f tag_name="$version" -f previous_tag_name="$prev_root" --jq .body
    else
      gh api "repos/${repo}/releases/generate-notes" -f tag_name="$version" --jq .body
    fi

    local module prev
    for module in "$MODULES_API_DIR" "$MODULES_CLI_DIR"; do
      prev="$(latest_semver_tag "${module}/" <<< "$tags")"
      [[ -n "$prev" ]] || continue
      printf '\n## Changes in `%s/`\n\n' "$module"
      GITHUB_REPOSITORY="$repo" "${REPO_ROOT}/scripts/generate-folder-release-notes.sh" \
        --folder "$module" --previous-tag "$prev" --new-tag "$version" --ref "${module}/${version}" \
        | sed '/^## What.s Changed$/d; /^\*\*Full Changelog/d'
    done
  } > "$outfile"
}
