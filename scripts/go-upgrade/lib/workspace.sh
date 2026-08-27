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

# Workspace management: cloning, clean state, module discovery.
# Sourced by upgrade.sh — do not execute directly.

# --- Module discovery with caching ---
declare -A MODULE_CACHE=()

discover_modules() {
  local entry="$1"
  local name
  name="$(repo_name "$entry")"

  # Return cached result if available
  if [[ -n "${MODULE_CACHE[$name]:-}" ]]; then
    echo "${MODULE_CACHE[$name]}"
    return
  fi

  local dir
  dir="$(ws)/${name}"
  if [[ ! -d "$dir" ]]; then
    err "Repository directory not found: ${dir}"
    return 1
  fi

  local result=""
  while IFS= read -r gomod; do
    local mod_dir="${gomod%/go.mod}"
    local rel="${mod_dir#$(ws)/}"
    if is_excluded_module "$rel"; then
      continue
    fi
    if [[ -n "$result" ]]; then
      result+=$'\n'
    fi
    result+="$mod_dir"
  done < <(find "$dir" -name go.mod -not -path '*/.git/*' | sort)

  MODULE_CACHE[$name]="$result"
  echo "$result"
}

# --- Workspace setup ---
ensure_workspace() {
  local workspace
  workspace="$(ws)"
  local need_clone=false
  while IFS= read -r entry; do
    IFS='|' read -r name url branch _ <<< "$entry"
    if [[ ! -d "${workspace}/${name}" ]]; then
      need_clone=true
      mkdir -p "$workspace"
      log "Cloning ${name} (branch: ${branch})..."
      git clone --branch "$branch" --single-branch "$url" "${workspace}/${name}"
    fi
  done < <(active_repos)
  if [[ "$need_clone" == false ]]; then
    log "Workspace exists, reusing: ${workspace}"
  fi
}

# Ensure each repo is on the correct branch with a clean working tree.
# Discards uncommitted changes from prior interrupted runs.
ensure_clean_state() {
  while IFS= read -r entry; do
    local name
    name="$(repo_name "$entry")"
    local dir
    dir="$(ws)/${name}"
    local base_branch
    base_branch="$(repo_base_branch "$entry")"

    [[ ! -d "$dir/.git" ]] && continue

    # Ensure we're on the base branch
    local current_branch
    current_branch=$(cd "$dir" && git rev-parse --abbrev-ref HEAD)
    if [[ "$current_branch" != "$base_branch" ]]; then
      warn "${name}: on branch '${current_branch}', switching to '${base_branch}'"
      (cd "$dir" && git checkout "$base_branch" 2>&1) || {
        record_failure "clean state: cannot checkout ${base_branch} in ${name}"
        continue
      }
    fi

    # Discard any uncommitted changes (leftover from prior run)
    if ! (cd "$dir" && git diff --quiet && git diff --cached --quiet); then
      warn "${name}: discarding uncommitted changes from prior run"
      (cd "$dir" && git reset --hard && git clean -fd) 2>&1 || true
    fi

    # Detect base branch pollution: local commits ahead of remote
    if ! (cd "$dir" && git fetch -q origin "$base_branch" 2>&1); then
      record_failure "clean state: cannot fetch origin/${base_branch} in ${name}"
      continue
    fi
    local ahead
    ahead=$(cd "$dir" && git rev-list --count "origin/${base_branch}..${base_branch}" 2>/dev/null) || ahead=0
    if [[ "$ahead" -gt 0 ]]; then
      err "${name}: base branch '${base_branch}' is ${ahead} commit(s) ahead of origin/${base_branch}"
      err "  This likely means a prior run committed directly to the base branch."
      err "  Fix manually: cd $(ws)/${name} && git reset --hard origin/${base_branch}"
      exit 1
    fi
  done < <(active_repos)
}

# --- Git status summary ---
show_git_status() {
  log "=== Git Status ==="
  while IFS= read -r entry; do
    local name
    name="$(repo_name "$entry")"
    local dir
    dir="$(ws)/${name}"
    if [[ ! -d "$dir/.git" ]]; then
      continue
    fi
    step "${name}"
    (cd "$dir" && git status --short) || true
    echo ""
  done < <(active_repos)
}
