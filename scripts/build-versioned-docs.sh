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

# build-versioned-docs.sh
#
# Builds versioned Hugo documentation for the kpt site using:
#   - Presentation (theme, layouts, config) from the current working tree
#   - Content from git tags for each released version
#
# Reads documentation/versions.json to determine which versions to build.
# Resolves tagPattern globs to the latest stable semver tag dynamically
# (pre-releases such as -alpha/-beta/-rc are ignored when a stable tag exists).
#
# Usage:
#   ./scripts/build-versioned-docs.sh [output-dir]
#
# Environment:
#   HUGO_ENV - Hugo environment (default: production)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DOCS_DIR="${REPO_ROOT}/documentation"
OUTPUT_DIR="${1:-${DOCS_DIR}/public}"
HUGO_ENV="${HUGO_ENV:-production}"

# Track temp directories for cleanup on exit
TEMP_DIRS=()
cleanup() {
  for d in "${TEMP_DIRS[@]:-}"; do
    rm -rf "${d}" 2>/dev/null || true
  done
  # Remove overlay file if left behind by a failed build
  rm -f "${DOCS_DIR}/config-versions-overlay.toml" 2>/dev/null || true
}
trap cleanup EXIT

# Ensure required tools are available
for cmd in hugo git node npx tar mktemp; do
  if ! command -v "${cmd}" &> /dev/null; then
    echo "ERROR: ${cmd} is required but not installed." >&2
    exit 1
  fi
done

VERSIONS_FILE="${DOCS_DIR}/versions.json"
if [[ ! -f "${VERSIONS_FILE}" ]]; then
  echo "ERROR: ${VERSIONS_FILE} not found." >&2
  exit 1
fi

# read_versions: parse and validate versions.json with a real JSON parser and
# emit one tab-separated record per entry: version<TAB>tagPattern<TAB>path<TAB>latestGA
# (tagPattern is the literal string "null" when the entry has no pattern). Using
# a JSON parser instead of line-oriented text tooling means the manifest can be
# reformatted (e.g. minified) without breaking the build. A malformed manifest
# fails the build loudly rather than silently producing an empty version list.
read_versions() {
  # shellcheck disable=SC2016  # single quotes are intentional: this is JS, not shell
  node -e '
    const fs = require("fs");
    let data;
    try {
      data = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
    } catch (e) {
      console.error("ERROR: failed to parse versions.json: " + e.message);
      process.exit(1);
    }
    if (!Array.isArray(data)) {
      console.error("ERROR: versions.json must be a JSON array.");
      process.exit(1);
    }
    for (const [i, e] of data.entries()) {
      if (!e || typeof e.version !== "string" || typeof e.path !== "string") {
        console.error(`ERROR: versions.json entry ${i} must have string "version" and "path".`);
        process.exit(1);
      }
      const tagPattern = (e.tagPattern === null || e.tagPattern === undefined) ? "null" : String(e.tagPattern);
      const latestGA = e.latestGA === true ? "true" : "false";
      process.stdout.write([e.version, tagPattern, e.path, latestGA].join("\t") + "\n");
    }
  ' "${VERSIONS_FILE}"
}

# Read (and validate) the manifest once up front so a malformed file fails
# before we start building anything.
VERSIONS_TSV="$(read_versions)"

echo "==> Building versioned docs"
echo "    Output: ${OUTPUT_DIR}"
echo "    Versions file: ${VERSIONS_FILE}"
echo ""

# Ensure we have all tags (Netlify may do shallow clones)
echo "==> Fetching tags..."
git -C "${REPO_ROOT}" fetch --unshallow 2>/dev/null || true
git -C "${REPO_ROOT}" fetch --tags --force 2>/dev/null || true
echo ""

# Clean output directory (with safety check)
if [[ -z "${OUTPUT_DIR}" || "${OUTPUT_DIR}" == "/" ]]; then
  echo "ERROR: OUTPUT_DIR is empty or root. Refusing to delete." >&2
  exit 1
fi
rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

# Emit [[params.versions]] TOML from the parsed manifest (single source of truth).
versions_toml() {
  while IFS=$'\t' read -r version _pattern path _latest_ga; do
    [[ -z "${version}" ]] && continue
    printf '[[params.versions]]\n  version = "%s"\n  url = "%s"\n\n' "${version}" "${path}"
  done <<< "${VERSIONS_TSV}"
}

# resolve_latest_tag: given a glob like "v1.0.*", return the newest tag.
# Prefer stable releases: if any non-pre-release tag matches, pick the highest
# of those; otherwise fall back to the highest pre-release. This avoids git's
# refname sort ranking "v1.0.0-beta.68" above the "v1.0.0" GA release.
resolve_latest_tag() {
  local pattern="$1"
  local stable
  stable="$(git -C "${REPO_ROOT}" tag --list "${pattern}" --sort=-v:refname \
    | grep -Ev -- '-(alpha|beta|rc|pre)' | head -1 || true)"
  if [[ -n "${stable}" ]]; then
    echo "${stable}"
    return
  fi
  git -C "${REPO_ROOT}" tag --list "${pattern}" --sort=-v:refname | head -1
}

# Build the main/latest version from the current working tree.
echo "==> Building latest (main) docs..."
(cd "${DOCS_DIR}" && { hugo mod clean 2>/dev/null || true; })

versions_toml > "${DOCS_DIR}/config-versions-overlay.toml"
(
  cd "${DOCS_DIR}"
  hugo --gc --minify \
    --environment "${HUGO_ENV}" \
    --destination "${OUTPUT_DIR}" \
    -b "/" \
    --config config.toml,config-versions-overlay.toml
)
rm -f "${DOCS_DIR}/config-versions-overlay.toml"
echo "    Done: latest -> /"
echo ""

# Build each tagged version (entries whose tagPattern is not null).
TAGGED_VERSIONS=$(while IFS=$'\t' read -r version pattern path latest_ga; do
  [[ -z "${version}" ]] && continue
  [[ "${pattern}" == "null" || -z "${pattern}" ]] && continue
  printf '%s\t%s\t%s\t%s\n' "${version}" "${pattern}" "${path}" "${latest_ga}"
done <<< "${VERSIONS_TSV}")

while IFS=$'\t' read -r VERSION PATTERN URL_PATH LATEST_GA; do
  [[ -z "${VERSION}" ]] && continue

  # Validate URL_PATH: must be absolute and free of traversal sequences.
  if [[ ! "${URL_PATH}" =~ ^/[a-zA-Z0-9._/-]*$ ]] || [[ "${URL_PATH}" == *".."* ]]; then
    echo "    ERROR: Invalid URL_PATH '${URL_PATH}' for ${VERSION}. Must be an absolute path without traversal." >&2
    continue
  fi

  TAG=$(resolve_latest_tag "${PATTERN}")
  if [[ -z "${TAG}" ]]; then
    echo "    WARNING: No tags matching '${PATTERN}' found, skipping ${VERSION}." >&2
    continue
  fi

  echo "==> Building ${VERSION} from tag ${TAG} (pattern: ${PATTERN})..."

  TEMP_DIR=$(mktemp -d)
  TEMP_DIRS+=("${TEMP_DIR}")
  TEMP_DOCS="${TEMP_DIR}/documentation"
  mkdir -p "${TEMP_DOCS}"

  # Presentation from current tree, content from tag.
  # Copy theme/layout/config assets from the working tree.
  cp -r "${DOCS_DIR}/assets" "${TEMP_DOCS}/" 2>/dev/null || true
  cp -r "${DOCS_DIR}/layouts" "${TEMP_DOCS}/" 2>/dev/null || true
  cp -r "${DOCS_DIR}/static" "${TEMP_DOCS}/" 2>/dev/null || true
  ln -s "${DOCS_DIR}/node_modules" "${TEMP_DOCS}/node_modules" 2>/dev/null || true
  cp "${DOCS_DIR}/go.mod" "${TEMP_DOCS}/" 2>/dev/null || true
  cp "${DOCS_DIR}/go.sum" "${TEMP_DOCS}/" 2>/dev/null || true
  cp "${DOCS_DIR}/config.toml" "${TEMP_DOCS}/"
  cp "${DOCS_DIR}/package.json" "${TEMP_DOCS}/" 2>/dev/null || true
  cp "${DOCS_DIR}/package-lock.json" "${TEMP_DOCS}/" 2>/dev/null || true
  cp "${DOCS_DIR}/postcss.config.js" "${TEMP_DOCS}/" 2>/dev/null || true

  # Extract version-specific documentation content from the tag.
  if ! git -C "${REPO_ROOT}" archive "${TAG}" -- documentation/content/ 2>/dev/null | \
    tar -x -C "${TEMP_DIR}" 2>/dev/null; then
    echo "    WARNING: Failed to extract documentation content from tag ${TAG}, skipping ${VERSION}." >&2
    continue
  fi

  # Verify content was actually extracted.
  if [[ ! -d "${TEMP_DOCS}/content" ]]; then
    echo "    WARNING: Tag ${TAG} has no documentation/content/ directory, skipping ${VERSION}." >&2
    continue
  fi

  # Use main's home page for consistent nav/layout across versions.
  cp "${DOCS_DIR}/content/en/_index.md" "${TEMP_DOCS}/content/en/_index.md" 2>/dev/null || true

  VERSION_OUTPUT="${OUTPUT_DIR}${URL_PATH}"
  mkdir -p "${VERSION_OUTPUT}"

  # The latest GA release is not archived, so it does not show the "no longer
  # actively maintained" banner. All other tagged versions are archived.
  ARCHIVED_VERSION="true"
  if [[ "${LATEST_GA}" == "true" ]]; then
    ARCHIVED_VERSION="false"
  fi

  # Config overlay marking this as an archived version, plus the shared
  # version dropdown entries.
  {
    echo "[params]"
    echo "archived_version = ${ARCHIVED_VERSION}"
    echo "version = \"${VERSION}\""
    echo "url_latest_version = \"/\""
    echo "version_menu = \"Releases\""
    echo ""
    versions_toml
  } > "${TEMP_DOCS}/config-version-override.toml"

  (
    cd "${TEMP_DOCS}"
    hugo --gc --minify \
      --environment "${HUGO_ENV}" \
      --destination "${VERSION_OUTPUT}" \
      -b "${URL_PATH}" \
      --config config.toml,config-version-override.toml
  )
  echo "    Done: ${VERSION} (${TAG}) -> ${URL_PATH}"
  echo ""
done <<< "${TAGGED_VERSIONS}"

# Build the search index once over the full combined output (latest + versions).
# A failure here (e.g. pagefind not installable in a restricted environment)
# should not fail the whole build: the site is fully rendered without it, only
# in-site search is degraded.
echo "==> Building search index (pagefind)..."
if ! npx -y pagefind --site "${OUTPUT_DIR}"; then
  echo "    WARNING: pagefind failed; search index not built. Site output is still complete." >&2
fi
echo ""

echo "==> All versions built successfully."
echo "    Output directory: ${OUTPUT_DIR}"
ls -la "${OUTPUT_DIR}"
