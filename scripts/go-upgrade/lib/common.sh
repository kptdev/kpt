#!/bin/bash
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

# Common utilities: logging, colors, failure tracking, helpers.
# Sourced by upgrade.sh — do not execute directly.

# --- Colors ---
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

# --- Logging ---
log()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()  { echo -e "${RED}[ERR]${NC} $*" >&2; }
step() { echo -e "${CYAN}[>>]${NC} $*"; }

# --- Failure tracking ---
FAILURES=()

record_failure() {
  FAILURES+=("$1")
  if [[ "$FAIL_FAST" == true ]]; then
    err "Fail-fast: $1"
    exit 1
  fi
}

report_failures() {
  if [[ ${#FAILURES[@]} -eq 0 ]]; then
    log "All operations completed successfully."
    return 0
  fi
  err "Failures (${#FAILURES[@]}):"
  for f in "${FAILURES[@]}"; do
    echo -e "  ${RED}✗${NC} $f"
  done
  return 1
}

# --- Workspace path ---
WORKSPACE=""
ws() {
  if [[ -z "$WORKSPACE" ]]; then
    WORKSPACE="${SCRIPT_DIR}/${WORKSPACE_DIR}"
  fi
  echo "$WORKSPACE"
}

# Relative path for display
rel_path() { echo "${1#$(ws)/}"; }

# --- Repo entry parsing ---
# Format: NAME|GIT_URL|BRANCH|PR_TARGET
repo_name() { IFS='|' read -r name _ _ _ <<< "$1"; echo "$name"; }

repo_url() { IFS='|' read -r _ url _ _ <<< "$1"; echo "$url"; }

repo_base_branch() { IFS='|' read -r _ _ branch _ <<< "$1"; echo "$branch"; }

repo_pr_target() {
  local entry="$1"
  IFS='|' read -r _ url _ pr_target <<< "$entry"
  if [[ -n "$pr_target" ]]; then
    echo "$pr_target"
  else
    echo "$url" | sed -E 's|.*[:/]([^/]+/[^/]+)\.git$|\1|'
  fi
}

# Return repos filtered by --repo flag
active_repos() {
  for entry in "${REPOS[@]}"; do
    local name
    name="$(repo_name "$entry")"
    if [[ -z "$FILTER_REPO" || "$name" == "$FILTER_REPO" ]]; then
      echo "$entry"
    fi
  done
}

# --- Exclusion checks ---
is_excluded_module() {
  local mod_path="$1"
  for pattern in "${EXCLUDE_MODULES[@]}"; do
    # shellcheck disable=SC2053
    if [[ "$mod_path" == $pattern ]]; then
      return 0
    fi
  done
  return 1
}

is_excluded_dep() {
  local repo="$1" dep="$2"
  for d in "${EXCLUDE_DEPS_GLOBAL[@]+"${EXCLUDE_DEPS_GLOBAL[@]}"}"; do
    [[ "$dep" == "$d" || "$dep" == "$d/"* ]] && return 0
  done
  local repo_exclusions="${EXCLUDE_DEPS_REPO[$repo]:-}"
  for d in $repo_exclusions; do
    [[ "$dep" == "$d" || "$dep" == "$d/"* ]] && return 0
  done
  return 1
}

# --- Dependency checks ---
check_deps() {
  command -v jq >/dev/null 2>&1 || { err "jq is required but not installed"; exit 1; }

  # gh is needed for both `--push` and the `push` subcommand.
  if [[ "$GIT_PUSH" == true || "$SUBCOMMAND" == "push" ]]; then
    command -v gh >/dev/null 2>&1 || { err "gh (GitHub CLI) is required for PR creation"; exit 1; }
  fi

  # Only require Go (and enforce exact version) when we will run Go tooling.
  case "$SUBCOMMAND" in
    go-version|cross-deps|all)
      command -v go >/dev/null 2>&1 || { err "go is required but not installed"; exit 1; }
      if [[ "$DRY_RUN" == false ]]; then
        local installed_go
        installed_go="$(go version | awk '{print $3}' | sed 's/^go//')"
        if [[ "$installed_go" != "$TARGET_GO_VERSION" ]]; then
          err "Installed Go version is ${installed_go}, but TARGET_GO_VERSION is ${TARGET_GO_VERSION}"
          err "Install Go ${TARGET_GO_VERSION} or update TARGET_GO_VERSION in config.env"
          exit 1
        fi
      fi
      ;;
    *)
      ;;
  esac
}
