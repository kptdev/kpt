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
# Release asset staging. Sourced by release.sh, do not execute directly.

# --- Subcommand: assets ---
# Copies everything GoReleaser built (dist/artifacts.json) that belongs on the
# GitHub release into OUTDIR, under the file names it would have uploaded:
# archives, linux packages, checksums and the plain binaries as kpt_OS_ARCH.
cmd_assets() {
  local dist="$1" outdir="$2"
  local artifacts="${dist}/artifacts.json"
  [[ -f "$artifacts" ]] || fatal "${artifacts} not found"
  mkdir -p "$outdir"

  local path
  while IFS= read -r path; do
    cp "$path" "$outdir/"
  done < <(jq -r '.[] | select(.type == "Archive" or .type == "Linux Package" or .type == "Checksum") | .path' "$artifacts")

  while IFS=$'\t' read -r path os arch; do
    cp "$path" "${outdir}/kpt_${os}_${arch}"
  done < <(jq -r '[.[] | select(.type == "Binary")] | unique_by(.path)[] | [.path, .goos, .goarch] | @tsv' "$artifacts")
  log "Staged $(find "$outdir" -type f | wc -l) release assets in ${outdir}"
}
