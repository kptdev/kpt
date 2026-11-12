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
# Minimal test helpers and a fake repository fixture for the release script
# tests. Sourced by the test-*.sh files, do not execute directly.

set -euo pipefail

TESTING_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RELEASE_SCRIPTS_DIR="$(cd "${TESTING_DIR}/.." && pwd)"
SCRIPTS_DIR="$(cd "${RELEASE_SCRIPTS_DIR}/.." && pwd)"

FAILURES=0
TESTS=0

pass() { TESTS=$((TESTS + 1)); }
fail() { TESTS=$((TESTS + 1)); FAILURES=$((FAILURES + 1)); echo "FAIL: $*"; }

assert_eq() { # WANT GOT MESSAGE
  if [[ "$1" == "$2" ]]; then pass; else fail "$3: want '$1', got '$2'"; fi
}
assert_contains() { # HAYSTACK NEEDLE MESSAGE
  if [[ "$1" == *"$2"* ]]; then pass; else fail "$3: '$2' not found in: $1"; fi
}
assert_not_contains() { # HAYSTACK NEEDLE MESSAGE
  if [[ "$1" != *"$2"* ]]; then pass; else fail "$3: '$2' unexpectedly found in: $1"; fi
}

finish() {
  if ((FAILURES > 0)); then
    echo "${FAILURES} of ${TESTS} assertions failed"
    exit 1
  fi
  echo "All ${TESTS} assertions passed"
}

# setup_tmp: creates a temp dir in TMP, isolated from the user's git config and
# Go module cache (the tests create fake kpt modules that must never end up in
# the real cache), and removes it on exit.
setup_tmp() {
  TMP="$(mktemp -d)"
  trap 'chmod -R u+w "$TMP" 2>/dev/null; rm -rf "$TMP"' EXIT
  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null
  export GIT_AUTHOR_NAME=Test GIT_AUTHOR_EMAIL=test@example.com
  export GIT_COMMITTER_NAME=Test GIT_COMMITTER_EMAIL=test@example.com
  export GOMODCACHE="${TMP}/gomodcache" GOTOOLCHAIN=local GOFLAGS=
}

# run_release ARGS...: runs release.sh in WORK, sets OUT (stdout+stderr) and RC.
run_release() {
  set +e
  OUT="$(cd "$WORK" && scripts/multimodule-release/release.sh "$@" 2>&1)"
  RC=$?
  set -e
}

# make_fixture: creates ORIGIN (bare) and WORK (clone) containing small fake api,
# root and cli modules with the real module paths and the release scripts. The
# commit is tagged and pushed as v1.0.0, api/v1.0.0 and cli/v1.0.0 on branch main.
make_fixture() {
  ORIGIN="${TMP}/origin.git"
  WORK="${TMP}/work"
  git init -q --bare -b main "$ORIGIN"
  git init -q -b main "$WORK"
  (
    cd "$WORK" || exit 1
    mkdir -p api cli scripts
    cp -r "$RELEASE_SCRIPTS_DIR" scripts/multimodule-release
    cp "${SCRIPTS_DIR}/generate-folder-release-notes.sh" scripts/

    printf 'module github.com/kptdev/kpt/api\n\ngo 1.21\n' > api/go.mod
    printf 'package api\n' > api/api.go
    printf 'module github.com/kptdev/kpt\n\ngo 1.21\n\nrequire github.com/kptdev/kpt/api v1.0.0\n' > go.mod
    printf 'package kpt\n\nimport _ "github.com/kptdev/kpt/api"\n' > kpt.go
    printf 'module github.com/kptdev/kpt/cli\n\ngo 1.21\n\nrequire (\n\tgithub.com/kptdev/kpt v1.0.0\n\tgithub.com/kptdev/kpt/api v1.0.0\n)\n' > cli/go.mod
    printf 'package main\n\nimport (\n\t_ "github.com/kptdev/kpt"\n\t_ "github.com/kptdev/kpt/api"\n)\n\nfunc main() {}\n' > cli/main.go

    git add -A
    git commit -q -m "initial"
    git tag v1.0.0 && git tag api/v1.0.0 && git tag cli/v1.0.0
    git remote add origin "$ORIGIN"
    git push -q origin main v1.0.0 api/v1.0.0 cli/v1.0.0
  )
}
