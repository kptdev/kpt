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

package cache

import (
	"context"
	"path/filepath"
	"testing"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/git"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	gogit "github.com/go-git/go-git/v5"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLatestPackages(t *testing.T) {
	ctx := context.Background()
	tarfile := filepath.Join("..", "git", "testdata", "nested-repository.tar")
	_, cachedGit := openRepositoryFromArchive(t, ctx, tarfile, "nested")

	wantLatest := map[string]string{
		"sample":                    "v2",
		"catalog/empty":             "v1",
		"catalog/gcp/bucket":        "v1",
		"catalog/namespace/basens":  "v3",
		"catalog/namespace/istions": "v3",
	}
	revisions, err := cachedGit.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("ListPackageRevisions failed: %v", err)
	}

	gotLatest := map[string]string{}
	for _, pr := range revisions {
		rev := pr.GetPackageRevision()

		if latest, ok := rev.Labels[api.LatestPackageRevisionKey]; ok {
			if got, want := latest, api.LatestPackageRevisionValue; got != want {
				t.Errorf("%s label value: got %q, want %q", api.LatestPackageRevisionKey, got, want)
				continue
			}

			if existing, ok := gotLatest[rev.Spec.PackageName]; ok {
				t.Errorf("Multiple latest package revisions for package %q: %q and %q",
					rev.Spec.PackageName, rev.Spec.Revision, existing)
			}

			// latest package
			gotLatest[rev.Spec.PackageName] = rev.Spec.Revision
		}
	}

	if !cmp.Equal(wantLatest, gotLatest) {
		t.Errorf("Latest package revisions differ (-want,+got): %s", cmp.Diff(wantLatest, gotLatest))
	}
}

func TestPublishedLatest(t *testing.T) {
	ctx := context.Background()
	tarfile := filepath.Join("..", "git", "testdata", "nested-repository.tar")
	_, cached := openRepositoryFromArchive(t, ctx, tarfile, "publish-test")

	revisions, err := cached.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{
		Package:  "catalog/gcp/bucket",
		Revision: "v2",
	})
	if err != nil {
		t.Fatalf("ListPackageRevisions failed: %v", err)
	}

	// Expect a single result
	if got, want := len(revisions), 1; got != want {
		t.Fatalf("ListPackageRevisions returned %d packages; want %d", got, want)
	}

	bucket := revisions[0]
	// Expect draft package
	if got, want := bucket.Lifecycle(), api.PackageRevisionLifecycleDraft; got != want {
		t.Fatalf("Bucket package lifecycle: got %s, want %s", got, want)
	}

	update, err := cached.UpdatePackageRevision(ctx, bucket)
	if err != nil {
		t.Fatalf("UpdatePackaeg(%s) failed: %v", bucket.Key(), err)
	}
	if err := update.UpdateLifecycle(ctx, api.PackageRevisionLifecyclePublished); err != nil {
		t.Fatalf("UpdateLifecycle failed; %v", err)
	}
	closed, err := update.Close(ctx)
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	resource := closed.GetPackageRevision()
	if got, ok := resource.Labels[api.LatestPackageRevisionKey]; !ok {
		t.Errorf("Label %s not found as expected", api.LatestPackageRevisionKey)
	} else if want := api.LatestPackageRevisionValue; got != want {
		t.Errorf("Latest label: got %s, want %s", got, want)
	}
}

func openRepositoryFromArchive(t *testing.T, ctx context.Context, tarfile, name string) (*gogit.Repository, *cachedRepository) {
	t.Helper()

	tempdir := t.TempDir()
	repo, address := git.ServeGitRepository(t, tarfile, tempdir)

	cache := NewCache(t.TempDir(), CacheOptions{})
	cachedGit, err := cache.OpenRepository(ctx, &v1alpha1.Repository{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.RepositoryGVK.Kind,
			APIVersion: v1alpha1.RepositoryGVK.GroupVersion().Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: v1alpha1.RepositorySpec{
			Deployment: false,
			Type:       v1alpha1.RepositoryTypeGit,
			Content:    v1alpha1.RepositoryContentPackage,
			Git: &v1alpha1.GitRepository{
				Repo: address,
			},
		},
	})
	if err != nil {
		t.Fatalf("OpenRepository(%q) of %q failed; %v", address, tarfile, err)
	}

	return repo, cachedGit
}
