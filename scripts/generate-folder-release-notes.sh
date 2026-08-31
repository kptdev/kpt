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

# Generate GitHub-style release notes scoped to a single folder.
#
# Calls the GitHub Release Notes API, then keeps only bullets whose linked PR
# also appears in git history for the folder (e.g. api/).
#
# Nested-module git tags are assumed to be FOLDER/vX.Y.Z (for example api/v1.0.0).
#
# Prerequisites: bash 3.2+, gh (authenticated), git.
#
# Example (kpt API module):
#   scripts/generate-folder-release-notes.sh \
#     --folder api \
#     --new-tag v1.0.1

set -euo pipefail

GITHUB_REPOSITORY=${GITHUB_REPOSITORY:-"kptdev/kpt"}

usage() {
  cat <<'EOF'
Usage: generate-folder-release-notes.sh [options]

Generate release notes for changes under FOLDER/ by filtering the output of
the GitHub Release Notes API. Git tags are FOLDER/vX.Y.Z (for example api/v1.0.0).

Options:
  --folder PATH           Folder to scope notes to (required, e.g. api)
  --previous-tag VERSION  Previous release version (e.g. v1.0.0 or api/v1.0.0);
                          default: latest semver tag for the folder
  --new-tag VERSION       New release version (required, e.g. v1.0.1)
  --ref REF               Commitish for the new release (default: HEAD)
  --attribution           Prepend a note that notes were auto-generated (uses gh user login)
  -o, --output FILE       Write notes to FILE instead of stdout
  -h, --help              Show this help

EOF
}

fail() {
  echo "generate-folder-release-notes.sh: $*" >&2
  exit 1
}

missing_value() {
  fail "missing value for $1"
}

# Strip a trailing slash so "api/" and "api" are equivalent.
normalize_folder() {
  local folder="$1"
  folder="${folder#./}"
  while [[ "$folder" == */ ]]; do
    folder="${folder%/}"
  done
  printf '%s' "$folder"
}

normalize_version() {
  local version="$1"
  local prefix="${folder_path}/"
  if [[ -n "$folder_path" && "$version" == "${prefix}"* ]]; then
    version="${version#"$prefix"}"
  fi
  if [[ "$version" =~ ^v ]]; then
    printf '%s' "$version"
  else
    printf 'v%s' "$version"
  fi
}

folder_tag() {
  local version
  version="$(normalize_version "$1")"
  printf '%s/%s' "$folder_path" "$version"
}

# Latest strict SemVer long tag for the folder (FOLDER/vMAJOR.MINOR.PATCH).
latest_folder_tag() {
  git tag -l "${folder_path}/v*" --sort=v:refname |
    grep -E -- '/v[0-9]+\.[0-9]+\.[0-9]+$' |
    tail -n 1 || true
}

folder_path=""
previous_version=""
new_version=""
ref="HEAD"
add_attribution=0
output_file=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --folder|--directory|--dir)
      [[ $# -ge 2 ]] || missing_value "$1"
      folder_path="$2"
      shift 2
      ;;
    --previous-tag|--prev-tag|--prev)
      [[ $# -ge 2 ]] || missing_value "$1"
      previous_version="$2"
      shift 2
      ;;
    --new-tag|--new|--next-tag|--next)
      [[ $# -ge 2 ]] || missing_value "$1"
      new_version="$2"
      shift 2
      ;;
    --ref|--target|--target-ref)
      [[ $# -ge 2 ]] || missing_value "$1"
      ref="$2"
      shift 2
      ;;
    --attribution)
      add_attribution=1
      shift
      ;;
    -o | --output)
      [[ $# -ge 2 ]] || missing_value "$1"
      output_file="$2"
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1 (try --help)"
      ;;
  esac
done

if [[ -z "$folder_path" ]]; then
  fail "--folder is required (try --help)"
fi
if [[ -z "$new_version" ]]; then
  fail "--new-tag is required"
fi

folder_path="$(normalize_folder "$folder_path")"
if [[ -z "$folder_path" ]]; then
  fail "--folder must be a non-empty path (e.g. api)"
fi
if [[ ! -d "$folder_path" ]]; then
  fail "folder not found: ${folder_path}/"
fi

if [[ -z "$previous_version" ]]; then
  prev_long="$(latest_folder_tag)"
  if [[ -z "$prev_long" ]]; then
    fail "no prior semver tag ${folder_path}/vMAJOR.MINOR.PATCH; pass --previous-tag explicitly"
  fi
  previous_version="${prev_long##*/}"
fi

previous_tag="$(folder_tag "$previous_version")"
new_tag="$(folder_tag "$new_version")"

extract_pr_number() {
  local line="$1"
  if [[ "$line" =~ pull/([0-9]+) ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
  fi
}

extract_pr_number_from_subject() {
  local subject="$1"
  if [[ "$subject" =~ \(#([0-9]+)\)$ ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
  elif [[ "$subject" =~ ^Merge[[:space:]]pull[[:space:]]request[[:space:]]#([0-9]+) ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
  fi
}

# Discover PRs that touched the folder via git history (newline-separated, sorted).
folder_pr_numbers="$(
  git log "${previous_tag}..${ref}" --format='%s' -- "${folder_path}/" |
    while IFS= read -r subject; do
      [[ -z "$subject" ]] && continue
      pr_number="$(extract_pr_number_from_subject "$subject")"
      [[ -n "$pr_number" ]] || continue
      printf '%s\n' "$pr_number"
    done | sort -nu
)"

pr_in_folder() {
  local pr="$1"
  [[ -n "$folder_pr_numbers" ]] || return 1
  printf '%s\n' "$folder_pr_numbers" | grep -Fqx "$pr"
}

filter_bullet_line() {
  local line="$1"
  local pr_number

  pr_number="$(extract_pr_number "$line")"
  [[ -n "$pr_number" ]] || return 1
  pr_in_folder "$pr_number"
}

parse_and_filter_notes() {
  local body="$1"
  local section=""
  local whats_changed=()
  local new_contributors=()
  local full_changelog=""

  while IFS= read -r line || [[ -n "$line" ]]; do
    case "$line" in
      "## What's Changed")
        section="whats_changed"
        continue
        ;;
      "## New Contributors")
        section="new_contributors"
        continue
        ;;
      "**Full Changelog"*)
        full_changelog="$line"
        section=""
        continue
        ;;
    esac

    if [[ "$line" == "* "* ]]; then
      if filter_bullet_line "$line"; then
        case "$section" in
          whats_changed) whats_changed+=("$line") ;;
          new_contributors) new_contributors+=("$line") ;;
        esac
      fi
    fi
  done <<< "$body"

  printf '%s\n' "## What's Changed"
  if [[ ${#whats_changed[@]} -eq 0 ]]; then
    printf "* No changes under \`%s/\` in this range.\n" "$folder_path"
  else
    printf '%s\n' "${whats_changed[@]}"
  fi

  if [[ ${#new_contributors[@]} -gt 0 ]]; then
    printf '\n%s\n' "## New Contributors"
    printf '%s\n' "${new_contributors[@]}"
  fi

  if [[ -n "$full_changelog" ]]; then
    printf '\n%s\n' "$full_changelog"
  fi
}

gh_user_login() {
  gh api user --jq .login 2>/dev/null || true
}

prepend_attribution() {
  local notes="$1"
  local login

  login="$(gh_user_login)"
  if [[ -n "$login" ]]; then
    printf "> **Note:** These release notes were auto-generated by @%s and include only changes under \`%s/\`.\n\n%s\n" \
      "$login" "$folder_path" "$notes"
  else
    printf "> **Note:** These release notes were auto-generated and include only changes under \`%s/\`.\n\n%s\n" \
      "$folder_path" "$notes"
  fi
}

notes_body="$(
  gh api "repos/${GITHUB_REPOSITORY}/releases/generate-notes" \
    -f tag_name="$new_tag" \
    -f previous_tag_name="$previous_tag" \
    -f target_commitish="$ref" \
    --jq .body
)"

filtered_notes="$(parse_and_filter_notes "$notes_body")"

if [[ "$add_attribution" -eq 1 ]]; then
  filtered_notes="$(prepend_attribution "$filtered_notes")"
fi

if [[ -n "$output_file" ]]; then
  printf '%s\n' "$filtered_notes" > "$output_file"
else
  printf '%s\n' "$filtered_notes"
fi
