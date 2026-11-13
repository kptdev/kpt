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

# Strict semantic version handling. Sourced by release.sh, do not execute directly.

SEMVER_RE='^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?$'

# is_semver VERSION: succeeds for MAJOR.MINOR.PATCH[-PRERELEASE] with optional
# "v" prefix. Build metadata is not allowed.
is_semver() {
  [[ "$1" =~ $SEMVER_RE ]]
}

# semver_cmp A B: prints -1, 0 or 1 following semver precedence rules.
semver_cmp() {
  local a="$1" b="$2"
  [[ "$a" =~ $SEMVER_RE ]] || fatal "invalid version: $a"
  local a_core=("${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}" "${BASH_REMATCH[3]}") a_pre="${BASH_REMATCH[5]}"
  [[ "$b" =~ $SEMVER_RE ]] || fatal "invalid version: $b"
  local b_core=("${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}" "${BASH_REMATCH[3]}") b_pre="${BASH_REMATCH[5]}"

  local i
  for i in 0 1 2; do
    if ((a_core[i] > b_core[i])); then echo 1; return; fi
    if ((a_core[i] < b_core[i])); then echo -1; return; fi
  done

  # A version without a prerelease is newer than one with a prerelease.
  if [[ -z "$a_pre" && -z "$b_pre" ]]; then echo 0; return; fi
  if [[ -z "$a_pre" ]]; then echo 1; return; fi
  if [[ -z "$b_pre" ]]; then echo -1; return; fi

  local a_ids b_ids
  IFS='.' read -r -a a_ids <<< "$a_pre"
  IFS='.' read -r -a b_ids <<< "$b_pre"
  local n=${#a_ids[@]}
  ((${#b_ids[@]} < n)) && n=${#b_ids[@]}
  for ((i = 0; i < n; i++)); do
    local x="${a_ids[i]}" y="${b_ids[i]}"
    if [[ "$x" == "$y" ]]; then continue; fi
    local x_num=0 y_num=0
    [[ "$x" =~ ^[0-9]+$ ]] && x_num=1
    [[ "$y" =~ ^[0-9]+$ ]] && y_num=1
    if ((x_num && y_num)); then
      if ((x > y)); then echo 1; else echo -1; fi
    elif ((x_num)); then
      echo -1 # numeric identifiers are lower than alphanumeric ones
    elif ((y_num)); then
      echo 1
    elif [[ "$x" > "$y" ]]; then
      echo 1
    else
      echo -1
    fi
    return
  done
  # All shared identifiers are equal: the longer prerelease is newer.
  if ((${#a_ids[@]} > ${#b_ids[@]})); then echo 1
  elif ((${#a_ids[@]} < ${#b_ids[@]})); then echo -1
  else echo 0; fi
}

# latest_semver_tag PREFIX: reads tag names on stdin and prints the highest
# one that is PREFIX followed by a semantic version (PREFIX is "" or "api/").
latest_semver_tag() {
  local prefix="$1" tag best=""
  while IFS= read -r tag; do
    [[ "$tag" == "$prefix"* ]] || continue
    local ver="${tag#"$prefix"}"
    [[ "$ver" == v* ]] || continue
    is_semver "$ver" || continue
    if [[ -z "$best" || "$(semver_cmp "$ver" "${best#"$prefix"}")" == 1 ]]; then
      best="$tag"
    fi
  done
  echo "$best"
}
