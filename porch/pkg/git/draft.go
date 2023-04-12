// Copyright 2022 The kpt Authors
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

package git

import (
	"context"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/go-git/go-git/v5/plumbing"
	"go.opentelemetry.io/otel/trace"
)

type gitPackageDraft struct {
	parent        *gitRepository // repo is repo containing the package
	path          string         // the path to the package from the repo root
	revision      string
	workspaceName v1alpha1.WorkspaceName
	updated       time.Time
	tasks         []v1alpha1.Task

	// New value of the package revision lifecycle
	lifecycle v1alpha1.PackageRevisionLifecycle

	// ref to the base of the package update commit chain (used for conditional push)
	base *plumbing.Reference

	// name of the branch where the changes will be pushed
	branch BranchName

	// Current HEAD of the package changes (commit sha)
	commit plumbing.Hash

	// Cached tree of the package itself, some descendent of commit.Tree()
	tree plumbing.Hash
}

var _ repository.PackageDraft = &gitPackageDraft{}

func (d *gitPackageDraft) UpdateResources(ctx context.Context, new *v1alpha1.PackageRevisionResources, change *v1alpha1.Task) error {
	ctx, span := tracer.Start(ctx, "gitPackageDraft::UpdateResources", trace.WithAttributes())
	defer span.End()

	return d.parent.UpdateDraftResources(ctx, d, new, change)
}

func (d *gitPackageDraft) UpdateLifecycle(ctx context.Context, new v1alpha1.PackageRevisionLifecycle) error {
	d.lifecycle = new
	return nil
}

// Finish round of updates.
func (d *gitPackageDraft) Close(ctx context.Context) (repository.PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "gitPackageDraft::Close", trace.WithAttributes())
	defer span.End()

	return d.parent.CloseDraft(ctx, d)
}
