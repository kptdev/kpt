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
# Tagging, bump commits and push. Sourced by release.sh, do not execute directly.

# Runs go commands so that the kpt module tags, which only exist locally at
# this point, can be resolved: the kpt repository URL is redirected to the
# local clone and the proxy and checksum database are bypassed for kpt modules.
# Only the environment of the invoked command is changed.
with_local_kpt() {
  GIT_CONFIG_COUNT=1 \
  GIT_CONFIG_KEY_0="url.file://${REPO_ROOT}/.git.insteadOf" \
  GIT_CONFIG_VALUE_0="https://github.com/kptdev/kpt" \
  GOPROXY=direct \
  GONOSUMDB="github.com/kptdev/*" \
  GOFLAGS=-mod=mod \
  "$@"
}

# bump_module DIR COMMIT_MESSAGE MODULE@VERSION...
bump_module() {
  local dir="$1" message="$2"
  shift 2
  local req
  for req in "$@"; do
    (cd "$dir" && go mod edit -require="$req")
  done
  (cd "$dir" && with_local_kpt go mod tidy)
  git add "${dir}/go.mod" "${dir}/go.sum"
  git diff --cached --quiet && fatal "${dir}/go.mod did not change when bumping to ${*}"
  git commit -q -s -m "$message"
}

# --- Subcommand: tag ---
cmd_tag() {
  local version="$1"
  is_semver "$version" || fatal "invalid version: $version"
  version="v${version#v}"

  git diff --quiet HEAD || fatal "Working tree has uncommitted changes to tracked files"
  log "Tagging ${API_MODULE} as api/${version}"
  git tag -a "api/${version}" -m "api ${version}"

  log "Updating the api requirement of the root module"
  bump_module . "Release ${version}: update api module to ${version}" "${API_MODULE}@${version}"
  git tag -a "${version}" -m "${version}"

  log "Updating the kpt and api requirements of the cli module"
  bump_module "$MODULES_CLI_DIR" "Release ${version}: update kpt and api modules to ${version}" \
    "${ROOT_MODULE}@${version}" "${API_MODULE}@${version}"
  git tag -a "cli/${version}" -m "cli ${version}"

  # The workflow checks afterwards that everything is pinned to tagged versions.
  (cd "$MODULES_CLI_DIR" && CGO_ENABLED=0 with_local_kpt go build ./...)
  log "Created tags api/${version}, ${version} and cli/${version}"
}

# --- Subcommand: push ---
cmd_push() {
  local version="v${1#v}" branch="$2"
  local refs=("HEAD:refs/heads/${branch}"
    "refs/tags/api/${version}" "refs/tags/${version}" "refs/tags/cli/${version}")
  if [[ "${DRY_RUN:-false}" == "true" ]]; then
    log "DRY_RUN: would run: git push --atomic origin ${refs[*]}"
    return
  fi
  git push --atomic origin "${refs[@]}"
}
