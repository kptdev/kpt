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

# Version upgrade subcommands: go-version, lint-version.
# Sourced by upgrade.sh — do not execute directly.

# --- Subcommand: go-version ---
cmd_go_version() {
  log "=== Upgrading Go version to ${TARGET_GO_VERSION} ==="

  while IFS= read -r entry; do
    local name
    name="$(repo_name "$entry")"
    step "Repository: ${name}"

    local modules
    modules="$(discover_modules "$entry")"

    while IFS= read -r mod_abs; do
      [[ -z "$mod_abs" ]] && continue
      local gomod="${mod_abs}/go.mod"
      local current
      current=$(grep -m1 '^go ' "$gomod" | awk '{print $2}')

      if [[ "$current" == "$TARGET_GO_VERSION" ]]; then
        continue
      fi

      if [[ "$DRY_RUN" == true ]]; then
        log "  [dry-run] $(rel_path "$mod_abs"): ${current} → ${TARGET_GO_VERSION}"
        continue
      fi

      sed -i.bak "s/^go .*/go ${TARGET_GO_VERSION}/" "$gomod" && rm -f "${gomod}.bak"
      log "  $(rel_path "$mod_abs"): ${current} → ${TARGET_GO_VERSION}"
    done <<< "$modules"

    if [[ "$DRY_RUN" == true ]]; then
      continue
    fi

    while IFS= read -r mod_abs; do
      [[ -z "$mod_abs" ]] && continue
      step "  verify: $(rel_path "$mod_abs")"
      verify_module "$mod_abs" || true
    done <<< "$modules"
  done < <(active_repos)
}

# --- Subcommand: lint-version ---
cmd_lint_version() {
  log "=== Upgrading golangci-lint version to ${TARGET_GOLANGCI_LINT_VERSION} ==="

  while IFS= read -r entry; do
    local name
    name="$(repo_name "$entry")"
    local makefile_rel="${LINT_MAKEFILES[$name]:-}"

    if [[ -z "$makefile_rel" ]]; then
      warn "${name}: no lint Makefile configured, skipping"
      continue
    fi

    local makefile="$(ws)/${name}/${makefile_rel}"

    if [[ ! -f "$makefile" ]]; then
      warn "${name}: ${makefile_rel} not found, skipping"
      continue
    fi

    local current
    current=$(grep -Eo 'GOLANGCI_LINT_VERSION[[:space:]]*[:?]?=[[:space:]]*[0-9.]+' "$makefile" | head -1 | sed -E 's/.*=[[:space:]]*//' || true)

    if [[ -z "$current" ]]; then
      warn "${name}: GOLANGCI_LINT_VERSION not found in ${makefile_rel}, skipping"
      continue
    fi

    if [[ "$current" == "$TARGET_GOLANGCI_LINT_VERSION" ]]; then
      log "  ${name}: already at ${TARGET_GOLANGCI_LINT_VERSION}"
      continue
    fi

    if [[ "$DRY_RUN" == true ]]; then
      log "  [dry-run] ${name}: ${current} → ${TARGET_GOLANGCI_LINT_VERSION}"
      continue
    fi

    sed -i.bak -E "s/(GOLANGCI_LINT_VERSION[[:space:]]*[:?]?=[[:space:]]*)[0-9.]+/\1${TARGET_GOLANGCI_LINT_VERSION}/" "$makefile" && rm -f "${makefile}.bak"
    log "  ${name}: ${current} → ${TARGET_GOLANGCI_LINT_VERSION}"
  done < <(active_repos)
}

# --- Subcommand: generate-docs ---
cmd_generate_docs() {
  log "=== Generating catalog documentation ==="

  local catalog_dir="$(ws)/krm-functions-catalog"

  if [[ ! -d "$catalog_dir" ]]; then
    record_failure "generate-docs: krm-functions-catalog not found in workspace"
    return
  fi

  # Respect --repo filter: skip if filtering to a different repo
  if [[ -n "$FILTER_REPO" && "$FILTER_REPO" != "krm-functions-catalog" ]]; then
    log "  skipped: generate-docs only applies to krm-functions-catalog"
    return
  fi

  if [[ "$DRY_RUN" == true ]]; then
    log "  [dry-run] would run: make generate-docs in krm-functions-catalog"
    (cd "$catalog_dir" && cd scripts/generate_docs && go run . generate --dry-run 2>&1) || true
    return
  fi

  if ! (cd "$catalog_dir" && make generate-docs 2>&1); then
    record_failure "generate-docs: make generate-docs failed"
    return
  fi

  log "  Done."
}
