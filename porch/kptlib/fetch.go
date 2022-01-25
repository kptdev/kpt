// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kptlib

import (
	"context"

	"github.com/GoogleContainerTools/kpt/internal/util/fetch"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"k8s.io/klog/v2"
)

type FetchResults struct {
	dir      string
	repoSpec GitRepoSpec
}

func (r *FetchResults) AbsPath() string {
	return r.dir
}

func (r *FetchResults) GitRepoSpec() GitRepoSpec {
	return r.repoSpec
}

type GitRepoSpec = git.RepoSpec

func Fetch(ctx context.Context, g *GitRepoSpec) (*FetchResults, error) {
	updated := *g
	// pr.Printf("Fetching upstream from %s@%s\n", kf.Upstream.Git.Repo, kf.Upstream.Git.Ref)
	klog.Infof("Fetching from %s@%s\n", updated.OrgRepo, updated.Ref)
	if err := fetch.ClonerUsingGitExec(ctx, &updated); err != nil {
		return nil, err // return errors.E(op, p.UniquePath, err)
	}
	results := &FetchResults{
		dir:      updated.AbsPath(),
		repoSpec: updated,
	}
	return results, nil
}
