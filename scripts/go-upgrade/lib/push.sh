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

# Push subcommand: create branch, commit, push, and raise PR.
# Sourced by upgrade.sh — do not execute directly.

# --- Subcommand: push ---
cmd_push() {
  log "=== Creating branches, committing, pushing, and raising PRs ==="

  local branch_name
  case "$SUBCOMMAND" in
    go-version)     branch_name="upgrade-go-${TARGET_GO_VERSION}-$(date +%Y%m%d)" ;;
    lint-version)   branch_name="upgrade-golangci-lint-${TARGET_GOLANGCI_LINT_VERSION}-$(date +%Y%m%d)" ;;
    cross-deps)     branch_name="update-cross-deps-$(date +%Y%m%d)" ;;
    generate-docs)  branch_name="generate-docs-$(date +%Y%m%d)" ;;
    all)            branch_name="upgrade-go-${TARGET_GO_VERSION}-deps-$(date +%Y%m%d)" ;;
  esac
  local -a created_prs=()

  while IFS= read -r entry; do
    local name
    name="$(repo_name "$entry")"
    local dir
    dir="$(ws)/${name}"
    local base_branch
    base_branch="$(repo_base_branch "$entry")"

    if [[ ! -d "$dir/.git" ]]; then
      warn "Skipping ${name}: not a git repository"
      continue
    fi

    step "Repository: ${name}"

    # Checkout base branch first
    if ! (cd "$dir" && git checkout "$base_branch" 2>&1); then
      record_failure "checkout ${base_branch}: ${name}"
      continue
    fi

    # Check if there are changes to commit
    if (cd "$dir" && git diff --quiet && git diff --cached --quiet); then
      log "  ${name}: no changes to commit, skipping"
      continue
    fi

    local target
    target="$(repo_pr_target "$entry")"

    # Create branch (fail if exists)
    if ! (cd "$dir" && git checkout -b "$branch_name" 2>&1); then
      record_failure "branch creation: ${name} (branch ${branch_name} may already exist)"
      continue
    fi

    # Stage modified tracked files and any new go.sum files
    (cd "$dir" && git add -u)
    (cd "$dir" && find . -name 'go.sum' -not -path '*/.git/*' -exec git add {} + 2>/dev/null) || true
    # Stage generated documentation files
    if [[ "$SUBCOMMAND" == "generate-docs" ]]; then
      (cd "$dir" && git add documentation/content/ 2>/dev/null) || true
    fi

    # Build commit message based on what was actually done
    local commit_msg="" pr_title="" pr_body_items=""
    case "$SUBCOMMAND" in
      go-version)
        commit_msg="Upgrade Go to ${TARGET_GO_VERSION}"
        pr_title="Upgrade Go to ${TARGET_GO_VERSION}"
        pr_body_items="- Go version bumped to \`${TARGET_GO_VERSION}\`"
        ;;
      lint-version)
        commit_msg="Upgrade golangci-lint to ${TARGET_GOLANGCI_LINT_VERSION}"
        pr_title="Upgrade golangci-lint to ${TARGET_GOLANGCI_LINT_VERSION}"
        pr_body_items="- golangci-lint version bumped to \`${TARGET_GOLANGCI_LINT_VERSION}\`"
        ;;
      cross-deps)
        commit_msg="Update cross-repository dependencies"
        pr_title="Update cross-repository dependencies"
        pr_body_items="- Cross-repository dependencies updated to latest"
        ;;
      generate-docs)
        commit_msg="Regenerate catalog documentation"
        pr_title="Regenerate catalog documentation"
        pr_body_items="- Hugo doc pages regenerated from function source"
        ;;
      all)
        commit_msg="Upgrade Go to ${TARGET_GO_VERSION} and update dependencies"
        pr_title="Upgrade Go to ${TARGET_GO_VERSION} and update dependencies"
        pr_body_items="- Go version bumped to \`${TARGET_GO_VERSION}\`
- golangci-lint version bumped to \`${TARGET_GOLANGCI_LINT_VERSION}\`
- Cross-repository dependencies updated to latest"
        ;;
    esac
    local verified_line=""
    if [[ "$SUBCOMMAND" != "generate-docs" ]]; then
      verified_line="
- All modules verified (go mod tidy, go fmt, go vet, go build)"
    fi
    commit_msg="${commit_msg}

${pr_body_items}${verified_line}"

    local pr_body_verified=""
    if [[ "$SUBCOMMAND" != "generate-docs" ]]; then
      pr_body_verified="
- All modules verified (\`go mod tidy\`, \`go fmt\`, \`go vet\`, \`go build\`)"
    fi
    local pr_body
    pr_body="## Description

${pr_body_items}${pr_body_verified}

## AI Disclosure

- [x] **I have used AI in the creation of this PR.**

Automated via go-upgrade script (AI-assisted development)."

    if ! (cd "$dir" && git commit --signoff -m "$commit_msg" 2>&1); then
      record_failure "commit: ${name}"
      continue
    fi

    log "  committed: ${branch_name}"

    # Push
    if ! (cd "$dir" && git push -u origin "$branch_name" 2>&1); then
      record_failure "push: ${name}"
      continue
    fi

    log "  pushed: ${branch_name}"

    # Verify branch has commits ahead of base before creating PR
    if (cd "$dir" && [[ "$(git rev-parse "origin/${base_branch}")" == "$(git rev-parse HEAD)" ]]); then
      warn "  ${name}: branch has no new commits vs origin/${base_branch}, skipping PR"
      continue
    fi

    # Determine head ref and repository IDs for cross-fork PR
    local origin_repo
    origin_repo=$(cd "$dir" && git remote get-url origin | sed -E 's|.*[:/]([^/]+/[^/]+)\.git$|\1|')

    log "  PR: ${origin_repo}:${branch_name} → ${target}:${base_branch}"
    log "  commits ahead: $(cd "$dir" && git log --oneline "origin/${base_branch}..HEAD" | wc -l)"

    # Allow GitHub to propagate the pushed branch
    sleep 5

    # Use GraphQL mutation with headRepositoryId for reliable cross-fork PRs
    local target_owner="${target%%/*}"
    local target_name="${target##*/}"
    local origin_owner="${origin_repo%%/*}"
    local origin_name="${origin_repo##*/}"

    local target_repo_id origin_repo_id
    target_repo_id=$(gh api graphql \
      -F owner="$target_owner" \
      -F name="$target_name" \
      -f query='query($owner: String!, $name: String!) { repository(owner: $owner, name: $name) { id } }' \
      --jq '.data.repository.id') || true
    origin_repo_id=$(gh api graphql \
      -F owner="$origin_owner" \
      -F name="$origin_name" \
      -f query='query($owner: String!, $name: String!) { repository(owner: $owner, name: $name) { id } }' \
      --jq '.data.repository.id') || true

    if [[ -z "$target_repo_id" || -z "$origin_repo_id" ]]; then
      record_failure "PR creation: ${name} (could not resolve repo IDs)"
      continue
    fi

    local pr_url
    if ! pr_url=$(gh api graphql \
      -F repoId="$target_repo_id" \
      -F baseRef="$base_branch" \
      -F headRef="$branch_name" \
      -F headRepoId="$origin_repo_id" \
      -F title="$pr_title" \
      -F body="$pr_body" \
      -f query='
      mutation($repoId: ID!, $baseRef: String!, $headRef: String!, $headRepoId: ID!, $title: String!, $body: String!) {
        createPullRequest(input: {
          repositoryId: $repoId,
          baseRefName: $baseRef,
          headRefName: $headRef,
          headRepositoryId: $headRepoId,
          title: $title,
          body: $body,
          draft: true
        }) {
          pullRequest { url }
        }
      }
    ' --jq '.data.createPullRequest.pullRequest.url' 2>&1); then
      err "    ${pr_url}"
      record_failure "PR creation: ${name}"
      continue
    fi

    created_prs+=("${name}: ${pr_url}")
    log "  PR created: ${pr_url}"
  done < <(active_repos)

  if [[ ${#created_prs[@]} -gt 0 ]]; then
    echo ""
    log "=== PRs Created ==="
    for pr in "${created_prs[@]}"; do
      log "  ${pr}"
    done
  fi
}
