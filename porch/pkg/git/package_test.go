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
	"path/filepath"
	"testing"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-cmp/cmp"
)

func (g GitSuite) TestLock(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "drafts-repository.tar")
	repo, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

	ctx := context.Background()
	const (
		repositoryName = "lock"
		namespace      = "default"
		deployment     = true
	)

	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    g.branch,
		Directory: "/",
	}, deployment, tempdir, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("Failed to list packages from %q: %v", tarfile, err)
	}

	wantRefs := map[repository.PackageRevisionKey]string{
		{Repository: repositoryName, Package: "empty", Revision: "v1", WorkspaceName: "v1"}:   "empty/v1",
		{Repository: repositoryName, Package: "basens", Revision: "v1", WorkspaceName: "v1"}:  "basens/v1",
		{Repository: repositoryName, Package: "basens", Revision: "v2", WorkspaceName: "v2"}:  "basens/v2",
		{Repository: repositoryName, Package: "istions", Revision: "v1", WorkspaceName: "v1"}: "istions/v1",
		{Repository: repositoryName, Package: "istions", Revision: "v2", WorkspaceName: "v2"}: "istions/v2",

		{Repository: repositoryName, Package: "basens", Revision: g.branch, WorkspaceName: v1alpha1.WorkspaceName(g.branch)}:  g.branch,
		{Repository: repositoryName, Package: "empty", Revision: g.branch, WorkspaceName: v1alpha1.WorkspaceName(g.branch)}:   g.branch,
		{Repository: repositoryName, Package: "istions", Revision: g.branch, WorkspaceName: v1alpha1.WorkspaceName(g.branch)}: g.branch,
	}

	for _, rev := range revisions {
		if rev.Lifecycle() != v1alpha1.PackageRevisionLifecyclePublished {
			continue
		}

		upstream, lock, err := rev.GetLock()
		if err != nil {
			t.Errorf("GetUpstreamLock(%q) failed: %v", rev.Key(), err)
		}
		if got, want := upstream.Type, v1.GitOrigin; got != want {
			t.Errorf("upstream.Type: got %s, want %s", got, want)
		}
		if got, want := lock.Type, v1.GitOrigin; got != want {
			t.Errorf("lock.Type: got %s, want %s", got, want)
		}

		key := rev.Key()
		wantRef, ok := wantRefs[key]
		if !ok {
			t.Errorf("Unexpected package found; %q", rev.Key())
		}

		type gitAddress struct {
			Repo, Directory, Ref string
		}

		// Check upstream values
		if got, want := (gitAddress{
			Repo:      upstream.Git.Repo,
			Directory: upstream.Git.Directory,
			Ref:       upstream.Git.Ref,
		}), (gitAddress{
			Repo:      address,
			Directory: key.Package,
			Ref:       wantRef,
		}); !cmp.Equal(want, got) {
			t.Errorf("Package upstream differs (-want,+got): %s", cmp.Diff(want, got))
		}

		// Check upstream lock values
		if got, want := (gitAddress{
			Repo:      lock.Git.Repo,
			Directory: lock.Git.Directory,
			Ref:       lock.Git.Ref,
		}), (gitAddress{
			Repo:      address,
			Directory: key.Package,
			Ref:       wantRef,
		}); !cmp.Equal(want, got) {
			t.Errorf("Package upstream lock differs (-want,+got): %s", cmp.Diff(want, got))
		}

		// Check the commit
		if commit, err := repo.ResolveRevision(plumbing.Revision(wantRef)); err != nil {
			t.Errorf("ResolveRevision(%q) failed: %v", wantRef, err)
		} else if got, want := lock.Git.Commit, commit.String(); got != want {
			t.Errorf("Commit: got %s, want %s", got, want)
		}
	}
}
