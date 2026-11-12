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

# Multimodule release utility: releases the api, root and cli modules with one
# locked version. See release/README.md for the full process.
#
# Usage: release.sh <subcommand> [args]
#
# Subcommands:
#   validate VERSION BRANCH   Validate the version/branch and print the normalized version
#   tag VERSION               Create the bump commits and the api/, root and cli/ tags locally
#   push VERSION BRANCH       Atomically push the branch and the three tags (DRY_RUN=true skips)
#   notes VERSION BRANCH OUTFILE
#                             Write the combined release notes for the changes up to BRANCH
#                             (the new tags do not need to be pushed yet)
#   assets DIST OUTDIR        Stage the GoReleaser artifacts to attach to the release
#
# Environment:
#   DRY_RUN                   When "true", `push` does not push anything
#   GITHUB_REPOSITORY         Repository for release notes (default: kptdev/kpt)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

source "${SCRIPT_DIR}/lib/common.sh"
source "${SCRIPT_DIR}/lib/semver.sh"
source "${SCRIPT_DIR}/lib/validate.sh"
source "${SCRIPT_DIR}/lib/tag.sh"
source "${SCRIPT_DIR}/lib/notes.sh"
source "${SCRIPT_DIR}/lib/assets.sh"

usage() {
  sed -n '/^# Usage:/,/^$/p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
  exit 2
}

main() {
  [[ $# -ge 1 ]] || usage
  local cmd="$1"
  shift
  cd "$REPO_ROOT"
  case "$cmd" in
    validate)     [[ $# -eq 2 ]] || usage; cmd_validate "$@" ;;
    tag)          [[ $# -eq 1 ]] || usage; cmd_tag "$@" ;;
    push)         [[ $# -eq 2 ]] || usage; cmd_push "$@" ;;
    notes)        [[ $# -eq 3 ]] || usage; cmd_notes "$@" ;;
    assets)       [[ $# -eq 2 ]] || usage; cmd_assets "$@" ;;
    *)            usage ;;
  esac
}

main "$@"
