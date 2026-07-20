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

# Cross-repository dependency upgrade subcommand.
# Sourced by upgrade.sh — do not execute directly.

# --- Module map: module_path -> repo_name ---
declare -A MOD_PATH_TO_REPO=()

build_module_map() {
  for entry in "${REPOS[@]}"; do
    local name
    name="$(repo_name "$entry")"
    [[ ! -d "$(ws)/${name}" ]] && continue
    while IFS= read -r mod_abs; do
      [[ -z "$mod_abs" ]] && continue
      local mod_path
      mod_path=$(grep -m1 '^module ' "${mod_abs}/go.mod" | awk '{print $2}')
      [[ -z "$mod_path" ]] && continue
      MOD_PATH_TO_REPO[$mod_path]="$name"
    done < <(discover_modules "$entry")
  done
}

# Find which repo owns a dependency path
find_dep_repo() {
  local dep="$1"
  for mod_path in "${!MOD_PATH_TO_REPO[@]}"; do
    if [[ "$dep" == "$mod_path" || "$dep" == "$mod_path/"* ]]; then
      echo "${MOD_PATH_TO_REPO[$mod_path]}"
      return
    fi
  done
}

# Resolve the latest stable version of a module.
# Tries: GitHub releases → go list (stable) → go list (all) → @latest pseudo-version.
resolve_latest_version() {
  local mod_abs="$1" dep="$2"
  local version=""

  # Try GitHub releases first (proxy may not have indexed the latest tag yet)
  local gh_owner_repo
  gh_owner_repo=$(echo "$dep" | sed -n 's|^github\.com/||p')
  if [[ -n "$gh_owner_repo" ]] && command -v gh >/dev/null 2>&1; then
    version=$(gh api "repos/${gh_owner_repo}/releases" --jq '.[0].tag_name' 2>/dev/null) || true
  fi

  # Fallback: Go module proxy — latest tagged version (exclude alpha and dev)
  if [[ -z "$version" || "$version" != v* ]]; then
    version=$(cd "$mod_abs" && GOWORK=off go list -m -versions "$dep" 2>/dev/null \
      | tr ' ' '\n' | grep -vE '\-(alpha|dev)' | tail -1) || true
  fi
  # Fallback: latest tagged version including pre-release
  if [[ -z "$version" || "$version" != v* ]]; then
    version=$(cd "$mod_abs" && GOWORK=off go list -m -versions "$dep" 2>/dev/null | awk '{print $NF}') || true
  fi
  # Fallback: pseudo-version via @latest
  if [[ -z "$version" || "$version" != v* ]]; then
    version=$(cd "$mod_abs" && GOWORK=off go list -m "${dep}@latest" 2>/dev/null | awk '{print $2}') || true
  fi
  echo "$version"
}

# --- Subcommand: cross-deps ---
cmd_cross_deps() {
  log "=== Upgrading cross-repository dependencies ==="

  # Build module path → repo name map once
  build_module_map

  while IFS= read -r entry; do
    local name
    name="$(repo_name "$entry")"
    step "Repository: ${name}"

    while IFS= read -r mod_abs; do
      [[ -z "$mod_abs" ]] && continue
      local gomod="${mod_abs}/go.mod"

      # Find cross-repo deps (direct + indirect)
      local cross_deps=()
      local all_deps
      all_deps=$(go mod edit -json "$gomod" | jq -r '.Require[]? | .Path')

      while IFS= read -r dep; do
        [[ -z "$dep" ]] && continue
        local dep_repo
        dep_repo="$(find_dep_repo "$dep")"
        if [[ -n "$dep_repo" && "$dep_repo" != "$name" ]]; then
          cross_deps+=("$dep")
        fi
      done <<< "$all_deps"

      if [[ ${#cross_deps[@]} -eq 0 ]]; then
        continue
      fi

      step "  cross-deps: $(rel_path "$mod_abs")"

      if [[ "$DRY_RUN" == true ]]; then
        for dep in "${cross_deps[@]}"; do
          if is_excluded_dep "$name" "$dep"; then
            log "    skip (excluded): ${dep}"
          else
            log "    ${dep} → (latest published)"
          fi
        done
        continue
      fi

      local upgrade_args=()
      for dep in "${cross_deps[@]}"; do
        if is_excluded_dep "$name" "$dep"; then
          log "    skip (excluded): ${dep}"
          continue
        fi
        local latest_version
        latest_version=$(resolve_latest_version "$mod_abs" "$dep")
        if [[ -z "$latest_version" || "$latest_version" != v* ]]; then
          warn "    ${dep}: no valid version found, skipping"
          continue
        fi
        log "    ${dep} → ${latest_version}"
        upgrade_args+=("${dep}@${latest_version}")
      done

      if [[ ${#upgrade_args[@]} -eq 0 ]]; then
        continue
      fi

      if ! (cd "$mod_abs" && GOWORK=off go get "${upgrade_args[@]}" 2>&1); then
        record_failure "cross-deps upgrade: $(rel_path "$mod_abs")"
        continue
      fi

      verify_module "$mod_abs" || true
    done < <(discover_modules "$entry")
  done < <(active_repos)
}
