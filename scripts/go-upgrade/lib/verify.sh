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

# Module verification: tidy, fmt, vet, build.
# Sourced by upgrade.sh — do not execute directly.

# Run go mod tidy + fmt + vet + build for a module.
# Records failures via record_failure and returns non-zero on error.
verify_module() {
  local abs_dir="$1"

  # Hugo modules have no Go packages — use hugo mod tidy instead
  local has_packages
  if ! has_packages=$(cd "$abs_dir" && GOWORK=off go list ./... 2>/dev/null); then
    record_failure "list: $(rel_path "$abs_dir")"
    return 1
  fi
  if [[ -z "$has_packages" ]]; then
    if command -v hugo >/dev/null 2>&1; then
      if ! (cd "$abs_dir" && hugo mod tidy 2>&1); then
        record_failure "hugo mod tidy: $(rel_path "$abs_dir")"
        return 1
      fi
    else
      warn "hugo not installed; skipping hugo mod tidy for $(rel_path "$abs_dir")"
    fi
    return 0
  fi

  if ! (cd "$abs_dir" && GOWORK=off go mod tidy 2>&1); then
    record_failure "tidy: $(rel_path "$abs_dir")"
    return 1
  fi
  if ! (cd "$abs_dir" && GOWORK=off go fmt ./... 2>&1); then
    record_failure "fmt: $(rel_path "$abs_dir")"
    return 1
  fi
  if ! (cd "$abs_dir" && GOWORK=off go vet ./... 2>&1); then
    record_failure "vet: $(rel_path "$abs_dir")"
    return 1
  fi
  if ! (cd "$abs_dir" && GOWORK=off go build ./... 2>&1); then
    record_failure "build: $(rel_path "$abs_dir")"
    return 1
  fi
  return 0
}
