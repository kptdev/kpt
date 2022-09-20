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

package engine

import (
	"context"
	"strings"
	"testing"

	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/cache"
	"github.com/GoogleContainerTools/kpt/porch/pkg/engine/fake"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/google/go-cmp/cmp"
)

func TestEdit(t *testing.T) {
	packageName := "foo-1234567890"
	packageRevision := &fake.PackageRevision{
		Name: packageName,
		Resources: &v1alpha1.PackageRevisionResources{
			Spec: v1alpha1.PackageRevisionResourcesSpec{
				PackageName:    packageName,
				Revision:       "v1",
				RepositoryName: "foo",
				Resources: map[string]string{
					kptfile.KptFileName: strings.TrimSpace(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: sample description
					`),
				},
			},
		},
	}
	repo := &fake.Repository{
		PackageRevisions: []repository.PackageRevision{
			packageRevision,
		},
	}
	cad := &fakeCaD{
		cache:      cache.NewCache("", cache.CacheOptions{}),
		repository: repo,
	}

	epm := editPackageMutation{
		task: &v1alpha1.Task{
			Type: "edit",
			Edit: &v1alpha1.PackageEditTaskSpec{
				Source: &v1alpha1.PackageRevisionRef{
					Name: packageName,
				},
			},
		},
		namespace:         "test-namespace",
		referenceResolver: &fakeReferenceResolver{},
		cad:               cad,
	}

	res, _, err := epm.Apply(context.Background(), repository.PackageResources{})
	if err != nil {
		t.Errorf("task apply failed: %v", err)
	}

	want := strings.TrimSpace(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: sample description
	`)
	got := strings.TrimSpace(res.Contents[kptfile.KptFileName])
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

// Implementation of the ReferenceResolver interface for testing.
type fakeReferenceResolver struct{}

func (f *fakeReferenceResolver) ResolveReference(ctx context.Context, namespace, name string, result Object) error {
	return nil
}

// Implementation of the engine.CaDEngine interface for testing.
type fakeCaD struct {
	cache      *cache.Cache
	repository repository.Repository
}

var _ CaDEngine = &fakeCaD{}

func (f *fakeCaD) ObjectCache() cache.ObjectCache {
	return f.cache.ObjectCache()
}

func (f *fakeCaD) OpenRepository(context.Context, *configapi.Repository) (repository.Repository, error) {
	return f.repository, nil
}

func (f *fakeCaD) CreatePackageRevision(context.Context, *configapi.Repository, *v1alpha1.PackageRevision) (repository.PackageRevision, error) {
	return nil, nil
}

func (f *fakeCaD) UpdatePackageRevision(_ context.Context, _ *configapi.Repository, _ repository.PackageRevision, _, _ *v1alpha1.PackageRevision) (repository.PackageRevision, error) {
	return nil, nil
}

func (f *fakeCaD) UpdatePackageResources(_ context.Context, _ *configapi.Repository, _ repository.PackageRevision, _, _ *v1alpha1.PackageRevisionResources) (repository.PackageRevision, error) {
	return nil, nil
}

func (f *fakeCaD) DeletePackageRevision(context.Context, *configapi.Repository, repository.PackageRevision) error {
	return nil
}

func (f *fakeCaD) ListFunctions(context.Context, *configapi.Repository) ([]repository.Function, error) {
	return []repository.Function{}, nil
}

func (f *fakeCaD) CreatePackage(context.Context, *configapi.Repository, *v1alpha1.Package) (repository.Package, error) {
	return nil, nil
}

func (f *fakeCaD) UpdatePackage(_ context.Context, _ *configapi.Repository, _ repository.Package, _, _ *v1alpha1.Package) (repository.Package, error) {
	return nil, nil
}

func (f *fakeCaD) DeletePackage(context.Context, *configapi.Repository, repository.Package) error {
	return nil
}
