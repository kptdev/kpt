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

# Go Upgrade Utility — multi-repo Go version and dependency upgrade tool.
#
# Usage: ./upgrade.sh <subcommand> [options]
#
# See README.md for full documentation.

# Require bash >= 4 (associative arrays are used throughout the utility).
if ((BASH_VERSINFO[0] < 4)); then
  echo "[ERR] bash >= 4 is required (found ${BASH_VERSION}). Please install a newer bash and ensure it's first in PATH." >&2
  exit 1
fi

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# --- Load configuration and library modules ---
source "${SCRIPT_DIR}/config.env"
source "${SCRIPT_DIR}/lib/common.sh"
source "${SCRIPT_DIR}/lib/workspace.sh"
source "${SCRIPT_DIR}/lib/verify.sh"
source "${SCRIPT_DIR}/lib/versions.sh"
source "${SCRIPT_DIR}/lib/deps.sh"
source "${SCRIPT_DIR}/lib/push.sh"

# --- Options ---
FAIL_FAST=true
DRY_RUN=false
GIT_PUSH=false
FILTER_REPO=""
SUBCOMMAND=""
PUSH_FOR=""

# --- Usage ---
usage() {
  cat <<EOF
Usage: $(basename "$0") <subcommand> [options]

Subcommands:
  go-version      Bump Go version in all go.mod files, tidy, and build
  lint-version    Bump golangci-lint version in all Makefiles
  cross-deps      Upgrade cross-repository dependencies to latest published version
  generate-docs   Generate/sync Hugo doc pages in krm-functions-catalog
  all             Run go-version + lint-version + cross-deps in sequence
  push            Create branch, commit, push, and raise PR for pending changes

Options:
  --repo=NAME   Run only against the specified repository
  --dry-run     Show what would change without modifying files
  --continue    Don't fail-fast; accumulate errors and report at end
  --push        After operations, create branch, commit, push, and raise PR
  --for=CMD     With 'push' subcommand: specify which upgrade was done
                (go-version, lint-version, cross-deps, generate-docs, all). Default: all

Environment:
  FORK_OWNER    Override fork owner (default: Nordix). Used to derive clone URLs.
                Example: FORK_OWNER=myuser ./upgrade.sh go-version

Configuration: config.env
EOF
  exit 1
}

# --- Parse args ---
parse_args() {
  if [[ $# -eq 0 ]]; then usage; fi

  for arg in "$@"; do
    case "$arg" in
      go-version|lint-version|cross-deps|generate-docs|all|push)
        if [[ -n "$SUBCOMMAND" && "$SUBCOMMAND" != "$arg" ]]; then
          err "Multiple subcommands provided: ${SUBCOMMAND} and ${arg}"
          usage
        fi
        SUBCOMMAND="$arg" ;;
      --dry-run)
        DRY_RUN=true ;;
      --continue)
        FAIL_FAST=false ;;
      --push)
        GIT_PUSH=true ;;
      --for=*)
        PUSH_FOR="${arg#--for=}" ;;
      --repo=*)
        FILTER_REPO="${arg#--repo=}" ;;
      -h|--help)
        usage ;;
      *)
        err "Unknown argument: $arg"
        usage ;;
    esac
  done

  if [[ -z "$SUBCOMMAND" ]]; then usage; fi

  # Validate --repo value
  if [[ -n "$FILTER_REPO" ]]; then
    local valid=false
    for entry in "${REPOS[@]}"; do
      if [[ "$(repo_name "$entry")" == "$FILTER_REPO" ]]; then
        valid=true
        break
      fi
    done
    if [[ "$valid" == false ]]; then
      err "Unknown repo: ${FILTER_REPO}"
      err "Available: $(for e in "${REPOS[@]}"; do repo_name "$e"; done | tr '\n' ' ')"
      exit 1
    fi
  fi

  # Validate --for value when push subcommand is used
  if [[ "$SUBCOMMAND" == "push" && -n "$PUSH_FOR" ]]; then
    case "$PUSH_FOR" in
      go-version|lint-version|cross-deps|generate-docs|all) ;;
      *)
        err "Invalid --for value: ${PUSH_FOR}"
        err "Valid values: go-version, lint-version, cross-deps, generate-docs, all"
        exit 1 ;;
    esac
  fi
}

# --- Main ---
main() {
  parse_args "$@"
  check_deps

  log "=== Go Upgrade Utility ==="
  log "Subcommand: ${SUBCOMMAND}"
  log "Target Go: ${TARGET_GO_VERSION}"
  log "Target golangci-lint: ${TARGET_GOLANGCI_LINT_VERSION}"
  log "Fork owner: ${FORK_OWNER}"
  if [[ "$DRY_RUN" == true ]]; then log "Mode: dry-run"; fi
  if [[ "$GIT_PUSH" == true ]]; then log "Mode: push enabled"; fi
  if [[ -n "$FILTER_REPO" ]]; then log "Repo filter: ${FILTER_REPO}"; fi
  echo ""

  # cross-deps needs all repos present to build the module-path→repo map,
  # so temporarily disable filtering for the clone step.
  # generate-docs only needs krm-functions-catalog.
  local orig_filter="$FILTER_REPO"
  if [[ "$SUBCOMMAND" == "cross-deps" || "$SUBCOMMAND" == "all" ]]; then
    FILTER_REPO=""
  elif [[ "$SUBCOMMAND" == "generate-docs" && -z "$FILTER_REPO" ]]; then
    FILTER_REPO="krm-functions-catalog"
  fi
  ensure_workspace
  FILTER_REPO="$orig_filter"
  ensure_clean_state

  local push_done=false
  case "$SUBCOMMAND" in
    go-version)    cmd_go_version ;;
    lint-version)  cmd_lint_version ;;
    cross-deps)    cmd_cross_deps ;;
    generate-docs) cmd_generate_docs ;;
    all)
      cmd_go_version
      cmd_lint_version
      cmd_cross_deps
      ;;
    push)
      # Standalone push: use --for to determine branch/commit naming
      SUBCOMMAND="${PUSH_FOR:-all}"
      cmd_push
      push_done=true
      ;;
  esac

  if [[ "$push_done" == false && "$GIT_PUSH" == true && "$DRY_RUN" == false ]]; then
    if [[ ${#FAILURES[@]} -gt 0 ]]; then
      warn "Skipping push: ${#FAILURES[@]} failure(s) during upgrade"
    else
      cmd_push
    fi
  fi

  echo ""
  if [[ "$DRY_RUN" == false ]]; then show_git_status; fi
  report_failures || exit 1
}

main "$@"
