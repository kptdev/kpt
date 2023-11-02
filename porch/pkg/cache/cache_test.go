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

package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	fakecache "github.com/GoogleContainerTools/kpt/porch/pkg/cache/fake"
	"github.com/GoogleContainerTools/kpt/porch/pkg/git"
	"github.com/GoogleContainerTools/kpt/porch/pkg/meta"
	fakemeta "github.com/GoogleContainerTools/kpt/porch/pkg/meta/fake"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	gogit "github.com/go-git/go-git/v5"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestLatestPackages(t *testing.T) {
	ctx := context.Background()
	testPath := filepath.Join("..", "git", "testdata")

	_, cachedGit := openRepositoryFromArchive(t, ctx, testPath, "nested")

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
		rev, err := pr.GetPackageRevision(ctx)
		if err != nil {
			t.Errorf("didn't expect error, but got %v", err)
		}

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
	testPath := filepath.Join("..", "git", "testdata")
	_, cached := openRepositoryFromArchive(t, ctx, testPath, "nested")

	revisions, err := cached.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{
		Package:       "catalog/gcp/bucket",
		WorkspaceName: "v2",
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
	resource, err := closed.GetPackageRevision(ctx)
	if err != nil {
		t.Errorf("didn't expect error, but got %v", err)
	}
	if got, ok := resource.Labels[api.LatestPackageRevisionKey]; !ok {
		t.Errorf("Label %s not found as expected", api.LatestPackageRevisionKey)
	} else if want := api.LatestPackageRevisionValue; got != want {
		t.Errorf("Latest label: got %s, want %s", got, want)
	}
}

func openRepositoryFromArchive(t *testing.T, ctx context.Context, testPath, name string) (*gogit.Repository, *cachedRepository) {
	t.Helper()

	tempdir := t.TempDir()
	tarfile := filepath.Join(testPath, fmt.Sprintf("%s-repository.tar", name))
	repo, address := git.ServeGitRepository(t, tarfile, tempdir)
	metadataStore := createMetadataStoreFromArchive(t, "", "")

	cache := NewCache(t.TempDir(), 60*time.Second, CacheOptions{
		MetadataStore:  metadataStore,
		ObjectNotifier: &fakecache.ObjectNotifier{},
	})
	cachedGit, err := cache.OpenRepository(ctx, &v1alpha1.Repository{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.TypeRepository.Kind,
			APIVersion: v1alpha1.TypeRepository.APIVersion(),
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

func createMetadataStoreFromArchive(t *testing.T, testPath, name string) meta.MetadataStore {
	t.Helper()

	f := filepath.Join("..", "git", "testdata", "nested-metadata.yaml")
	c, err := os.ReadFile(f)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Error reading metadata file found for repository %s", name)
	}
	if os.IsNotExist(err) {
		return &fakemeta.MemoryMetadataStore{
			Metas: []meta.PackageRevisionMeta{},
		}
	}

	var metas []meta.PackageRevisionMeta
	if err := yaml.Unmarshal(c, &metas); err != nil {
		t.Fatalf("Error unmarshalling metadata file for repository %s", name)
	}

	return &fakemeta.MemoryMetadataStore{
		Metas: metas,
	}
}
