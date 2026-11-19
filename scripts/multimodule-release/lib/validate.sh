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
# Input validation. Sourced by release.sh, do not execute directly.

# --- Subcommand: validate ---
# Prints the normalized version (with "v" prefix) on stdout.
cmd_validate() {
  local version="$1" branch="$2"

  is_semver "$version" || fatal "'${version}' is not a valid semantic version (expected MAJOR.MINOR.PATCH[-PRERELEASE])"
  version="v${version#v}"

  git ls-remote --exit-code --heads origin "$branch" >/dev/null 2>&1 \
    || fatal "Branch '${branch}' not found in remote 'origin'"

  local tags tag
  tags="$(remote_tags)"
  for tag in "${version}" "api/${version}" "cli/${version}"; do
    if grep -Fqx -- "$tag" <<< "$tags"; then
      fatal "Tag '${tag}' already exists"
    fi
  done

  # The new version must be newer than every existing release of every module.
  local prefix latest
  for prefix in "" "api/" "cli/"; do
    latest="$(latest_semver_tag "$prefix" <<< "$tags")"
    [[ -n "$latest" ]] || continue
    if [[ "$(semver_cmp "$version" "${latest#"$prefix"}")" != 1 ]]; then
      fatal "Version ${version} is not newer than existing tag '${latest}'"
    fi
  done

  log "Version ${version} is valid and newer than all existing releases"
  echo "$version"
}
