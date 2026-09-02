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

repos=(
  "https://github.com/kptdev/kpt"
  "https://github.com/kptdev/porch"
  "https://github.com/kptdev/krm-functions-catalog"
  "https://github.com/kptdev/krm-functions-sdk"
  "https://github.com/kptdev/kpt-backstage-plugins"
  "https://github.com/kptdev/kpt-samples"
)

function clone_repos () {
  repo_dir=$1

  for repo in "${repos[@]}"
  do
    git clone "$repo" > /dev/null 2>&1
  done
}

function create_pr_table () {
  echo "## PRs to be reviewed"
  echo ""
  echo "Generated on $(TZ=UTC date)"
  echo ""

  for repo in "${repos[@]}"
  do
    repo_heading=$(echo "$repo" | awk -F'/' '{printf("### [%s](https://%s/%s/%s/pulls)\n", $5, $3, $4, $5)}')
    kpt_repo=$(echo "$repo" | sed 's/.*\///')
    pushd "$kpt_repo" > /dev/null || exit

    echo "$repo_heading"
    echo ""
    echo "| Number | Title | Author | Draft | Updated | Comment |"
    echo "|-|-|-|-|-|-|"

    gh pr ls -L 200 --json title,author,isDraft,updatedAt,url | jq -r '.[] | "| \(.url) | \(.title) | \(.author.login) | \(.isDraft) | \(.updatedAt) | |"' || exit
    popd > /dev/null || exit
    echo ""
  done
}

function create_release_table () {
  echo "## Releases in kptdev"
  echo ""
  echo "Generated on $(TZ=UTC date)"
  echo ""

  for repo in "${repos[@]}"
  do
    repo_heading=$(echo "$repo" | awk -F'/' '{printf("### [%s](https://%s/%s/%s/releases)\n", $5, $3, $4, $5)}')
    kpt_repo=$(echo "$repo" | sed 's/.*\///')
    pushd "$kpt_repo" > /dev/null || exit

    echo "$repo_heading"
    echo ""
    echo "| Tag | Name | Latest | Pre Release | Published |"
    echo "|-|-|-|-|-|"

    gh release ls -L 1000 --exclude-drafts --json tagName,name,isLatest,isPrerelease,publishedAt | jq -r '.[] | "| \(.tagName) | \(.name) | \(.isLatest) | \(.isPrerelease) | \(.publishedAt) | |" ' | grep ' 202[4-6]-' || exit
    popd > /dev/null || exit
    echo ""
  done
}

repo_dir=$(mktemp -d)
trap 'rm -rf "$repo_dir"' EXIT

pushd "$repo_dir" > /dev/null || exit

clone_repos "$repo_dir"
create_pr_table "$repo_dir"
create_release_table "$repo_dir"

popd > /dev/null || exit
