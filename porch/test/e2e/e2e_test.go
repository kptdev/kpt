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

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/google/go-cmp/cmp"
	coreapi "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	testBlueprintsRepo = "https://github.com/platkrm/test-blueprints.git"
	kptRepo            = "https://github.com/GoogleContainerTools/kpt.git"
)

func TestE2E(t *testing.T) {
	Run(&PorchSuite{}, t)
}

func Run(suite interface{}, t *testing.T) {
	sv := reflect.ValueOf(suite)
	st := reflect.TypeOf(suite)
	ctx := context.Background()

	t.Run(st.Elem().Name(), func(t *testing.T) {
		var ts *TestSuite = sv.Elem().FieldByName("TestSuite").Addr().Interface().(*TestSuite)

		ts.T = t
		if init, ok := suite.(Initializer); ok {
			init.Initialize(ctx)
		}

		for i, max := 0, st.NumMethod(); i < max; i++ {
			m := st.Method(i)
			if strings.HasPrefix(m.Name, "Test") {
				t.Run(m.Name, func(t *testing.T) {
					ts.T = t
					m.Func.Call([]reflect.Value{sv, reflect.ValueOf(ctx)})
				})
			}
		}
	})
}

type PorchSuite struct {
	TestSuite
	config GitConfig
}

var _ Initializer = &PorchSuite{}

func (p *PorchSuite) Initialize(ctx context.Context) {
	p.TestSuite.Initialize(ctx)
	p.config = p.CreateGitRepo()
}

func (t *PorchSuite) TestGitRepository(ctx context.Context) {
	// Register the repository as 'git'
	t.registerMainGitRepositoryF(ctx, "git")

	// Create Package Revision
	pr := &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "test-bucket",
			Revision:       "v1",
			RepositoryName: "git",
			Tasks: []porchapi.Task{
				{
					Type: "clone",
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							Type: "git",
							Git: &porchapi.GitPackage{
								Repo:      "https://github.com/GoogleCloudPlatform/blueprints.git",
								Ref:       "bucket-blueprint-v0.4.3",
								Directory: "catalog/bucket",
							},
						},
					},
				},
				{
					Type: "eval",
					Eval: &porchapi.FunctionEvalTaskSpec{
						Image: "gcr.io/kpt-fn/set-namespace:v0.4.1",
						ConfigMap: map[string]string{
							"namespace": "bucket-namespace",
						},
					},
				},
			},
		},
	}
	t.CreateF(ctx, pr)

	// Get package resources
	var resources porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &resources)

	bucket, ok := resources.Spec.Resources["bucket.yaml"]
	if !ok {
		t.Errorf("'bucket.yaml' not found among package resources")
	}
	node, err := yaml.Parse(bucket)
	if err != nil {
		t.Errorf("yaml.Parse(\"bucket.yaml\") failed: %v", err)
	}
	if got, want := node.GetNamespace(), "bucket-namespace"; got != want {
		t.Errorf("StorageBucket namespace: got %q, want %q", got, want)
	}
}

func (t *PorchSuite) TestGitRepositoryWithReleaseTags(ctx context.Context) {
	t.registerGitRepositoryF(ctx, kptRepo, "kpt-repo", "package-examples")

	var list porchapi.PackageRevisionList
	t.ListF(ctx, &list, client.InNamespace(t.namespace))
}

func (t *PorchSuite) TestCloneFromUpstream(ctx context.Context) {
	// Register Upstream Repository
	t.registerGitRepositoryF(ctx, testBlueprintsRepo, "test-blueprints", "")

	var list porchapi.PackageRevisionList
	t.ListE(ctx, &list, client.InNamespace(t.namespace))

	basens := MustFindPackageRevision(t.T, &list, repository.PackageRevisionKey{Repository: "test-blueprints", Package: "basens", Revision: "v1"})

	// Register the repository as 'downstream'
	t.registerMainGitRepositoryF(ctx, "downstream")

	// Create PackageRevision from upstream repo
	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "istions",
			Revision:       "v1",
			RepositoryName: "downstream",
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeClone,
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							UpstreamRef: &porchapi.PackageRevisionRef{
								Name: basens.Name,
							},
						},
					},
				},
			},
		},
	}

	t.CreateF(ctx, pr)

	// Get istions resources
	var istions porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &istions)

	kptfile := t.ParseKptfileF(&istions)

	if got, want := kptfile.Name, "istions"; got != want {
		t.Errorf("istions package Kptfile.metadata.name: got %q, want %q", got, want)
	}
	if kptfile.UpstreamLock == nil {
		t.Fatalf("istions package upstreamLock is missing")
	}
	if kptfile.UpstreamLock.Git == nil {
		t.Errorf("istions package upstreamLock.git is missing")
	}
	if kptfile.UpstreamLock.Git.Commit == "" {
		t.Errorf("isions package upstreamLock.gkti.commit is missing")
	}

	// Remove commit from comparison
	got := kptfile.UpstreamLock
	got.Git.Commit = ""

	want := &kptfilev1.UpstreamLock{
		Type: kptfilev1.GitOrigin,
		Git: &kptfilev1.GitLock{
			Repo:      testBlueprintsRepo,
			Directory: "basens",
			Ref:       "basens/v1",
		},
	}
	if !cmp.Equal(want, got) {
		t.Errorf("unexpected upstreamlock returned (-want, +got) %s", cmp.Diff(want, got))
	}

	// Check Upstream
	if got, want := kptfile.Upstream, (&kptfilev1.Upstream{
		Type: kptfilev1.GitOrigin,
		Git: &kptfilev1.Git{
			Repo:      testBlueprintsRepo,
			Directory: "basens",
			Ref:       "basens/v1",
		},
	}); !cmp.Equal(want, got) {
		t.Errorf("unexpected upstream returned (-want, +got) %s", cmp.Diff(want, got))
	}
}

func (t *PorchSuite) TestInitEmptyPackage(ctx context.Context) {
	// Create a new package via init, no task specified
	const (
		repository  = "git"
		packageName = "empty-package"
		revision    = "v1"
		description = "empty-package description"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a new package (via init)
	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "empty-package",
			Revision:       "v1",
			RepositoryName: repository,
		},
	}
	t.CreateF(ctx, pr)

	// Get the package
	var newPackage porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &newPackage)

	kptfile := t.ParseKptfileF(&newPackage)
	if got, want := kptfile.Name, "empty-package"; got != want {
		t.Fatalf("New package name: got %q, want %q", got, want)
	}
	if got, want := kptfile.Info, (&kptfilev1.PackageInfo{
		Description: description,
	}); !cmp.Equal(want, got) {
		t.Fatalf("unexpected %s/%s package info (-want, +got) %s", newPackage.Namespace, newPackage.Name, cmp.Diff(want, got))
	}
}

func (t *PorchSuite) TestInitTaskPackage(ctx context.Context) {
	const (
		repository  = "git"
		packageName = "new-package"
		revision    = "v1"
		description = "New Package"
		site        = "https://kpt.dev/new-package"
	)
	keywords := []string{"test"}

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a new package (via init)
	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "new-package",
			Revision:       "v1",
			RepositoryName: repository,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeInit,
					Init: &porchapi.PackageInitTaskSpec{
						Description: description,
						Keywords:    keywords,
						Site:        site,
					},
				},
			},
		},
	}
	t.CreateF(ctx, pr)

	// Get the package
	var newPackage porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &newPackage)

	kptfile := t.ParseKptfileF(&newPackage)
	if got, want := kptfile.Name, "new-package"; got != want {
		t.Fatalf("New package name: got %q, want %q", got, want)
	}
	if got, want := kptfile.Info, (&kptfilev1.PackageInfo{
		Site:        site,
		Description: description,
		Keywords:    keywords,
	}); !cmp.Equal(want, got) {
		t.Fatalf("unexpected %s/%s package info (-want, +got) %s", newPackage.Namespace, newPackage.Name, cmp.Diff(want, got))
	}
}

func (t *PorchSuite) TestCloneIntoDeploymentRepository(ctx context.Context) {
	const downstreamRepository = "deployment"
	const downstreamPackage = "istions"
	const downstreamRevision = "v2"

	// Register the deployment repository
	t.registerMainGitRepositoryF(ctx, downstreamRepository, withDeployment())

	// Register the upstream repository
	t.registerGitRepositoryF(ctx, testBlueprintsRepo, "test-blueprints", "")

	var upstreamPackages porchapi.PackageRevisionList
	t.ListE(ctx, &upstreamPackages, client.InNamespace(t.namespace))
	upstreamPackage := MustFindPackageRevision(t.T, &upstreamPackages, repository.PackageRevisionKey{
		Repository: "test-blueprints",
		Package:    "basens",
		Revision:   "v1",
	})

	// Create PackageRevision from upstream repo
	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    downstreamPackage,
			Revision:       downstreamRevision,
			RepositoryName: downstreamRepository,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeClone,
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							UpstreamRef: &porchapi.PackageRevisionRef{
								Name: upstreamPackage.Name, // Package to be cloned
							},
						},
					},
				},
			},
		},
	}
	t.CreateF(ctx, pr)

	// Get istions resources
	var istions porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &istions)

	kptfile := t.ParseKptfileF(&istions)

	if got, want := kptfile.Name, "istions"; got != want {
		t.Errorf("istions package Kptfile.metadata.name: got %q, want %q", got, want)
	}
	if kptfile.UpstreamLock == nil {
		t.Fatalf("istions package upstreamLock is missing")
	}
	if kptfile.UpstreamLock.Git == nil {
		t.Errorf("istions package upstreamLock.git is missing")
	}
	if kptfile.UpstreamLock.Git.Commit == "" {
		t.Errorf("isions package upstreamLock.gkti.commit is missing")
	}

	// Remove commit from comparison
	got := kptfile.UpstreamLock
	got.Git.Commit = ""

	want := &kptfilev1.UpstreamLock{
		Type: kptfilev1.GitOrigin,
		Git: &kptfilev1.GitLock{
			Repo:      testBlueprintsRepo,
			Directory: "basens",
			Ref:       "basens/v1",
		},
	}
	if !cmp.Equal(want, got) {
		t.Errorf("unexpected upstreamlock returned (-want, +got) %s", cmp.Diff(want, got))
	}

	// Check Upstream
	if got, want := kptfile.Upstream, (&kptfilev1.Upstream{
		Type: kptfilev1.GitOrigin,
		Git: &kptfilev1.Git{
			Repo:      testBlueprintsRepo,
			Directory: "basens",
			Ref:       "basens/v1",
		},
	}); !cmp.Equal(want, got) {
		t.Errorf("unexpected upstream returned (-want, +got) %s", cmp.Diff(want, got))
	}

	// Check generated context
	var configmap coreapi.ConfigMap
	t.FindAndDecodeF(&istions, "package-context.yaml", &configmap)
	if got, want := configmap.Name, "kptfile.kpt.dev"; got != want {
		t.Errorf("package context name: got %s, want %s", got, want)
	}
	if got, want := configmap.Data["name"], "istions"; got != want {
		t.Errorf("package context 'data.name': got %s, want %s", got, want)
	}
}

// Test will initialize an empty package, update its resources, adding a function
// to the Kptfile's pipeline, and then check that the package was re-rendered.
func (t *PorchSuite) TestUpdateResources(ctx context.Context) {
	const (
		repository  = "re-render-test"
		packageName = "simple-package"
		revision    = "v3"
	)

	t.registerMainGitRepositoryF(ctx, repository)

	// Create a new package (via init)
	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    packageName,
			Revision:       revision,
			RepositoryName: repository,
		},
	}
	t.CreateF(ctx, pr)

	// Get the package resources
	var newPackage porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &newPackage)

	// Add function into a pipeline
	kptfile := t.ParseKptfileF(&newPackage)
	if kptfile.Pipeline == nil {
		kptfile.Pipeline = &kptfilev1.Pipeline{}
	}
	kptfile.Pipeline.Mutators = append(kptfile.Pipeline.Mutators, kptfilev1.Function{
		Image: "gcr.io/kpt-fn/set-annotations:v0.1.4",
		ConfigMap: map[string]string{
			"color": "red",
			"fruit": "apple",
		},
		Name: "set-annotations",
	})
	t.SaveKptfileF(&newPackage, kptfile)

	// Add a new resource
	filename := filepath.Join("testdata", "update-resources", "add-config-map.yaml")
	cm, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read ConfigMap from %q: %v", filename, err)
	}
	newPackage.Spec.Resources["config-map.yaml"] = string(cm)
	t.UpdateF(ctx, &newPackage)

	updated, ok := newPackage.Spec.Resources["config-map.yaml"]
	if !ok {
		t.Fatalf("Updated config map config-map.yaml not found")
	}

	golden := filepath.Join("testdata", "update-resources", "want-config-map.yaml")
	if diff := t.CompareGoldenFileYAML(golden, updated); diff != "" {
		t.Errorf("Unexpected updated confg map contents: (-want,+got): %s", diff)
	}
}

func (t *PorchSuite) TestFunctionRepository(ctx context.Context) {
	t.CreateF(ctx, &configapi.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "function-repository",
			Namespace: t.namespace,
		},
		Spec: configapi.RepositorySpec{
			Description: "Test Function Repository",
			Type:        configapi.RepositoryTypeOCI,
			Content:     configapi.RepositoryContentFunction,
			Oci: &configapi.OciRepository{
				Registry: "gcr.io/kpt-fn",
			},
		},
	})

	t.Cleanup(func() {
		t.DeleteL(ctx, &configapi.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "function-repository",
				Namespace: t.namespace,
			},
		})
	})

	list := &porchapi.FunctionList{}
	t.ListE(ctx, list, client.InNamespace(t.namespace))

	if got := len(list.Items); got == 0 {
		t.Errorf("Found no functions in gcr.io/kpt-fn repository; expected at least one")
	}
}

func (t *PorchSuite) TestPublicGitRepository(ctx context.Context) {
	t.registerGitRepositoryF(ctx, testBlueprintsRepo, "demo-blueprints", "")

	var list porchapi.PackageRevisionList
	t.ListE(ctx, &list, client.InNamespace(t.namespace))

	if got := len(list.Items); got == 0 {
		t.Errorf("Found no package revisions in %s; expected at least one", testBlueprintsRepo)
	}
}

func (t *PorchSuite) TestProposeApprove(ctx context.Context) {
	const (
		repository      = "lifecycle"
		packageName     = "test-package"
		packageRevision = "v1"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a new package (via init)
	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    packageName,
			Revision:       packageRevision,
			RepositoryName: repository,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeInit,
					Init: &porchapi.PackageInitTaskSpec{},
				},
			},
		},
	}
	t.CreateF(ctx, pr)

	var pkg porchapi.PackageRevision
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &pkg)

	// Propose the package revision to be finalized
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkg)

	var proposed porchapi.PackageRevision
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &proposed)

	if got, want := proposed.Spec.Lifecycle, porchapi.PackageRevisionLifecycleProposed; got != want {
		t.Fatalf("Proposed package lifecycle value: got %s, want %s", got, want)
	}

	// Approve using Update should fail.
	proposed.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	if err := t.client.Update(ctx, &proposed); err == nil {
		t.Fatalf("Finalization of a package via Update unexpectedly succeeded")
	}

	// Approve the package
	proposed.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	approved := t.UpdateApprovalF(ctx, &proposed, metav1.UpdateOptions{})
	if got, want := approved.Spec.Lifecycle, porchapi.PackageRevisionLifecyclePublished; got != want {
		t.Fatalf("Approved package lifecycle value: got %s, want %s", got, want)
	}
}

func (t *PorchSuite) TestDeleteDraft(ctx context.Context) {
	const (
		repository  = "delete-draft"
		packageName = "test-delete-draft"
		revision    = "v1"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	created := t.createPackageDraftF(ctx, repository, packageName, revision)

	// Check the package exists
	var draft porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &draft)

	// Delete the package
	t.DeleteE(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      created.Name,
		},
	})

	t.mustNotExist(ctx, &draft)
}

func (t *PorchSuite) TestDeleteProposed(ctx context.Context) {
	const (
		repository  = "delete-proposed"
		packageName = "test-delete-proposed"
		revision    = "v1"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	created := t.createPackageDraftF(ctx, repository, packageName, revision)

	// Check the package exists
	var pkg porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Propose the package revision to be finalized
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkg)

	// Delete the package
	t.DeleteE(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      created.Name,
		},
	})

	t.mustNotExist(ctx, &pkg)
}

func (t *PorchSuite) TestDeleteFinal(ctx context.Context) {
	const (
		repository  = "delete-final"
		packageName = "test-delete-final"
		revision    = "v1"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	created := t.createPackageDraftF(ctx, repository, packageName, revision)

	// Check the package exists
	var pkg porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Propose the package revision to be finalized
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkg)

	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	t.UpdateApprovalF(ctx, &pkg, metav1.UpdateOptions{})

	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Delete the package
	t.DeleteE(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      created.Name,
		},
	})

	t.mustNotExist(ctx, &pkg)
}

func (t *PorchSuite) TestDeleteAndRecreate(ctx context.Context) {
	const (
		repository  = "delete-and-recreate"
		packageName = "test-delete-and-recreate"
		revision    = "v1"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	created := t.createPackageDraftF(ctx, repository, packageName, revision)

	// Check the package exists
	var pkg porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Propose the package revision to be finalized
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkg)

	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	t.UpdateApprovalF(ctx, &pkg, metav1.UpdateOptions{})

	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Delete the package
	t.DeleteE(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      created.Name,
		},
	})

	t.mustNotExist(ctx, &pkg)

	// Recreate the package with the same name and revision
	created = t.createPackageDraftF(ctx, repository, packageName, revision)

	// Check the package exists
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Ensure that there is only one init task in the package revision history
	foundInitTask := false
	for _, task := range pkg.Spec.Tasks {
		if task.Type == porchapi.TaskTypeInit {
			if foundInitTask {
				t.Fatalf("found two init tasks in recreated package revision")
			}
			foundInitTask = true
		}
	}
	t.Logf("successfully recreated package revision %q", packageName)
}

func (t *PorchSuite) TestCloneLeadingSlash(ctx context.Context) {
	const (
		repository  = "clone-ls"
		packageName = "test-clone-ls"
		revision    = "v1"
	)

	t.registerMainGitRepositoryF(ctx, repository)

	// Clone the package. Use leading slash in the directory (regression test)
	new := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    packageName,
			Revision:       revision,
			RepositoryName: repository,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeClone,
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							Type: porchapi.RepositoryTypeGit,
							Git: &porchapi.GitPackage{
								Repo:      "https://github.com/platkrm/test-blueprints",
								Ref:       "basens/v1",
								Directory: "/basens",
							},
						},
						Strategy: porchapi.ResourceMerge,
					},
				},
			},
		},
	}

	t.CreateF(ctx, new)

	var pr porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: new.Name}, &pr)
}

func (t *PorchSuite) TestPackageUpdate(ctx context.Context) {
	const (
		gitRepository = "package-update"
	)

	t.registerGitRepositoryF(ctx, testBlueprintsRepo, "test-blueprints", "")

	var list porchapi.PackageRevisionList
	t.ListE(ctx, &list, client.InNamespace(t.namespace))

	basensV1 := MustFindPackageRevision(t.T, &list, repository.PackageRevisionKey{Repository: "test-blueprints", Package: "basens", Revision: "v1"})
	basensV2 := MustFindPackageRevision(t.T, &list, repository.PackageRevisionKey{Repository: "test-blueprints", Package: "basens", Revision: "v2"})

	// Register the repository as 'downstream'
	t.registerMainGitRepositoryF(ctx, gitRepository)

	// Create PackageRevision from upstream repo
	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "testns",
			Revision:       "v1",
			RepositoryName: gitRepository,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeClone,
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							UpstreamRef: &porchapi.PackageRevisionRef{
								Name: basensV1.Name,
							},
						},
					},
				},
			},
		},
	}

	t.CreateF(ctx, pr)

	var revisionResources porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &revisionResources)

	filename := filepath.Join("testdata", "update-resources", "add-config-map.yaml")
	cm, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read ConfigMap from %q: %v", filename, err)
	}
	revisionResources.Spec.Resources["config-map.yaml"] = string(cm)
	t.UpdateF(ctx, &revisionResources)

	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, pr)

	upstream := pr.Spec.Tasks[0].Clone.Upstream.DeepCopy()
	upstream.UpstreamRef.Name = basensV2.Name
	pr.Spec.Tasks = append(pr.Spec.Tasks, porchapi.Task{
		Type: porchapi.TaskTypeUpdate,
		Update: &porchapi.PackageUpdateTaskSpec{
			Upstream: *upstream,
		},
	})

	t.UpdateE(ctx, pr, &client.UpdateOptions{})

	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &revisionResources)

	if _, found := revisionResources.Spec.Resources["resourcequota.yaml"]; !found {
		t.Errorf("Updated package should contain 'resourcequota.yaml` file")
	}

}

func (t *PorchSuite) TestRegisterRepository(ctx context.Context) {
	const (
		repository = "register"
	)
	t.registerMainGitRepositoryF(ctx, repository,
		withContent(configapi.RepositoryContentPackage),
		withType(configapi.RepositoryTypeGit),
		withDeployment())

	var repo configapi.Repository
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      repository,
	}, &repo)

	if got, want := repo.Spec.Content, configapi.RepositoryContentPackage; got != want {
		t.Errorf("Repo Content: got %q, want %q", got, want)
	}
	if got, want := repo.Spec.Type, configapi.RepositoryTypeGit; got != want {
		t.Errorf("Repo Type: got %q, want %q", got, want)
	}
	if got, want := repo.Spec.Deployment, true; got != want {
		t.Errorf("Repo Deployment: got %t, want %t", got, want)
	}
}

func (t *PorchSuite) TestBuiltinFunctionEvaluator(ctx context.Context) {
	// Register the repository as 'git-fn'
	t.registerMainGitRepositoryF(ctx, "git-builtin-fn")

	// Create Package Revision
	pr := &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "test-builtin-fn-bucket",
			Revision:       "v1",
			RepositoryName: "git-builtin-fn",
			Tasks: []porchapi.Task{
				{
					Type: "clone",
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							Type: "git",
							Git: &porchapi.GitPackage{
								Repo:      "https://github.com/GoogleCloudPlatform/blueprints.git",
								Ref:       "bucket-blueprint-v0.4.3",
								Directory: "catalog/bucket",
							},
						},
					},
				},
				{
					Type: "eval",
					Eval: &porchapi.FunctionEvalTaskSpec{
						//
						Image: "gcr.io/kpt-fn/starlark:v0.4.3",
						ConfigMap: map[string]string{
							"source": `for resource in ctx.resource_list["items"]:
  resource["metadata"]["annotations"]["foo"] = "bar"`,
						},
					},
				},
				{
					Type: "eval",
					Eval: &porchapi.FunctionEvalTaskSpec{
						Image: "gcr.io/kpt-fn/set-namespace:v0.4.1",
						ConfigMap: map[string]string{
							"namespace": "bucket-namespace",
						},
					},
				},
				// TODO: add test for apply-replacements, we can't do it now because FunctionEvalTaskSpec doesn't allow
				// non-ConfigMap functionConfig due to a code generator issue when dealing with unstructured.
			},
		},
	}
	t.CreateF(ctx, pr)

	// Get package resources
	var resources porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &resources)

	bucket, ok := resources.Spec.Resources["bucket.yaml"]
	if !ok {
		t.Errorf("'bucket.yaml' not found among package resources")
	}
	node, err := yaml.Parse(bucket)
	if err != nil {
		t.Errorf("yaml.Parse(\"bucket.yaml\") failed: %v", err)
	}
	if got, want := node.GetNamespace(), "bucket-namespace"; got != want {
		t.Errorf("StorageBucket namespace: got %q, want %q", got, want)
	}
	annotations := node.GetAnnotations()
	if val, found := annotations["foo"]; !found || val != "bar" {
		t.Errorf("StorageBucket annotations should contain foo=bar, but got %v", annotations)
	}
}

func (t *PorchSuite) TestExecFunctionEvaluator(ctx context.Context) {
	// Register the repository as 'git-fn'
	t.registerMainGitRepositoryF(ctx, "git-fn")

	// Create Package Revision
	pr := &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "test-fn-bucket",
			Revision:       "v1",
			RepositoryName: "git-fn",
			Tasks: []porchapi.Task{
				{
					Type: "clone",
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							Type: "git",
							Git: &porchapi.GitPackage{
								Repo:      "https://github.com/GoogleCloudPlatform/blueprints.git",
								Ref:       "bucket-blueprint-v0.4.3",
								Directory: "catalog/bucket",
							},
						},
					},
				},
				{
					Type: "eval",
					Eval: &porchapi.FunctionEvalTaskSpec{
						Image: "gcr.io/kpt-fn/starlark:v0.3.0",
						ConfigMap: map[string]string{
							"source": `# set the namespace on all resources
for resource in ctx.resource_list["items"]:
  # mutate the resource
  resource["metadata"]["namespace"] = "bucket-namespace"`,
						},
					},
				},
				{
					Type: "eval",
					Eval: &porchapi.FunctionEvalTaskSpec{
						Image: "gcr.io/kpt-fn/set-annotations:v0.1.4",
						ConfigMap: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
		},
	}
	t.CreateF(ctx, pr)

	// Get package resources
	var resources porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &resources)

	bucket, ok := resources.Spec.Resources["bucket.yaml"]
	if !ok {
		t.Errorf("'bucket.yaml' not found among package resources")
	}
	node, err := yaml.Parse(bucket)
	if err != nil {
		t.Errorf("yaml.Parse(\"bucket.yaml\") failed: %v", err)
	}
	if got, want := node.GetNamespace(), "bucket-namespace"; got != want {
		t.Errorf("StorageBucket namespace: got %q, want %q", got, want)
	}
	annotations := node.GetAnnotations()
	if val, found := annotations["foo"]; !found || val != "bar" {
		t.Errorf("StorageBucket annotations should contain foo=bar, but got %v", annotations)
	}
}

func (t *PorchSuite) TestPodFunctionEvaluatorWithDistrolessImage(ctx context.Context) {
	if t.local {
		t.Skipf("Skipping due to not having pod evalutor in local mode")
	}

	t.registerMainGitRepositoryF(ctx, "git-fn-distroless")

	// Create Package Revision
	pr := &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "test-fn-redis-bucket",
			Revision:       "v1",
			RepositoryName: "git-fn-distroless",
			Tasks: []porchapi.Task{
				{
					Type: "clone",
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							Type: "git",
							Git: &porchapi.GitPackage{
								Repo:      "https://github.com/GoogleCloudPlatform/blueprints.git",
								Ref:       "redis-bucket-blueprint-v0.3.2",
								Directory: "catalog/redis-bucket",
							},
						},
					},
				},
				{
					Type: "patch",
					Patch: &porchapi.PackagePatchTaskSpec{
						Patches: []porchapi.PatchSpec{
							{
								File: "configmap.yaml",
								Contents: `apiVersion: v1
kind: ConfigMap
metadata:
  name: kptfile.kpt.dev
data:
  name: bucket-namespace
`,
								PatchType: porchapi.PatchTypeCreateFile,
							},
						},
					},
				},
				{
					Type: "eval",
					Eval: &porchapi.FunctionEvalTaskSpec{
						// This image is a mirror of gcr.io/cad-demo-sdk/set-namespace@sha256:462e44020221e72e3eb337ee59bc4bc3e5cb50b5ed69d377f55e05bec3a93d11
						// which uses gcr.io/distroless/base-debian11:latest as the base image.
						Image: "gcr.io/kpt-fn-demo/set-namespace:v0.1.0",
					},
				},
			},
		},
	}
	t.CreateF(ctx, pr)

	// Get package resources
	var resources porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &resources)

	bucket, ok := resources.Spec.Resources["bucket.yaml"]
	if !ok {
		t.Errorf("'bucket.yaml' not found among package resources")
	}
	node, err := yaml.Parse(bucket)
	if err != nil {
		t.Errorf("yaml.Parse(\"bucket.yaml\") failed: %v", err)
	}
	if got, want := node.GetNamespace(), "bucket-namespace"; got != want {
		t.Errorf("StorageBucket namespace: got %q, want %q", got, want)
	}
}

func (t *PorchSuite) TestPodEvaluator(ctx context.Context) {
	if t.local {
		t.Skipf("Skipping due to not having pod evalutor in local mode")
	}

	const (
		generateFolderImage = "gcr.io/kpt-fn/generate-folders:v0.1.1" // This function is a TS based function.
		setAnnotationsImage = "gcr.io/kpt-fn/set-annotations:v0.1.3"  // set-annotations:v0.1.3 is an older version that porch maps it neither to built-in nor exec.
	)

	// Register the repository as 'git-fn'
	t.registerMainGitRepositoryF(ctx, "git-fn-pod")

	// Create Package Revision
	pr := &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "test-fn-pod-hierarchy",
			Revision:       "v1",
			RepositoryName: "git-fn-pod",
			Tasks: []porchapi.Task{
				{
					Type: "clone",
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							Type: "git",
							Git: &porchapi.GitPackage{
								Repo:      "https://github.com/GoogleCloudPlatform/blueprints.git",
								Ref:       "783380ce4e6c3f21e9e90055b3a88bada0410154",
								Directory: "catalog/hierarchy/simple",
							},
						},
					},
				},
				// Testing pod evaluator with TS function
				{
					Type: "eval",
					Eval: &porchapi.FunctionEvalTaskSpec{
						Image: generateFolderImage,
					},
				},
				// Testing pod evaluator with golang function
				{
					Type: "eval",
					Eval: &porchapi.FunctionEvalTaskSpec{
						Image: setAnnotationsImage,
						ConfigMap: map[string]string{
							"test-key": "test-val",
						},
					},
				},
			},
		},
	}
	t.CreateF(ctx, pr)

	// Get package resources
	var resources porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &resources)

	counter := 0
	for name, obj := range resources.Spec.Resources {
		if strings.HasPrefix(name, "hierarchy/") {
			counter++
			node, err := yaml.Parse(obj)
			if err != nil {
				t.Errorf("failed to parse Folder object: %v", err)
			}
			if node.GetAnnotations()["test-key"] != "test-val" {
				t.Errorf("Folder should contain annotation `test-key:test-val`, the annotations we got: %v", node.GetAnnotations())
			}
		}
	}
	if counter != 4 {
		t.Errorf("expected 4 Folder objects, but got %v", counter)
	}

	// Get the fn runner pods and delete them.
	podList := &coreapi.PodList{}
	t.ListF(ctx, podList, client.InNamespace("porch-fn-system"))
	for _, pod := range podList.Items {
		img := pod.Spec.Containers[0].Image
		if img == generateFolderImage || img == setAnnotationsImage {
			t.DeleteF(ctx, &pod)
		}
	}

	// Create another Package Revision
	pr2 := &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "test-fn-pod-hierarchy",
			Revision:       "v2",
			RepositoryName: "git-fn-pod",
			Tasks: []porchapi.Task{
				{
					Type: "clone",
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							Type: "git",
							Git: &porchapi.GitPackage{
								Repo:      "https://github.com/GoogleCloudPlatform/blueprints.git",
								Ref:       "783380ce4e6c3f21e9e90055b3a88bada0410154",
								Directory: "catalog/hierarchy/simple",
							},
						},
					},
				},
				// Testing pod evaluator with TS function
				{
					Type: "eval",
					Eval: &porchapi.FunctionEvalTaskSpec{
						Image: generateFolderImage,
					},
				},
				// Testing pod evaluator with golang function
				{
					Type: "eval",
					Eval: &porchapi.FunctionEvalTaskSpec{
						Image: setAnnotationsImage,
						ConfigMap: map[string]string{
							"new-test-key": "new-test-val",
						},
					},
				},
			},
		},
	}
	t.CreateF(ctx, pr2)

	// Get package resources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr2.Name,
	}, &resources)

	counter = 0
	for name, obj := range resources.Spec.Resources {
		if strings.HasPrefix(name, "hierarchy/") {
			counter++
			node, err := yaml.Parse(obj)
			if err != nil {
				t.Errorf("failed to parse Folder object: %v", err)
			}
			if node.GetAnnotations()["new-test-key"] != "new-test-val" {
				t.Errorf("Folder should contain annotation `test-key:test-val`, the annotations we got: %v", node.GetAnnotations())
			}
		}
	}
	if counter != 4 {
		t.Errorf("expected 4 Folder objects, but got %v", counter)
	}
}

func (t *PorchSuite) TestPodEvaluatorWithFailure(ctx context.Context) {
	if t.local {
		t.Skipf("Skipping due to not having pod evalutor in local mode")
	}

	t.registerMainGitRepositoryF(ctx, "git-fn-pod-failure")

	// Create Package Revision
	pr := &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "test-fn-pod-bucket",
			Revision:       "v1",
			RepositoryName: "git-fn-pod-failure",
			Tasks: []porchapi.Task{
				{
					Type: "clone",
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							Type: "git",
							Git: &porchapi.GitPackage{
								Repo:      "https://github.com/GoogleCloudPlatform/blueprints.git",
								Ref:       "bucket-blueprint-v0.4.3",
								Directory: "catalog/bucket",
							},
						},
					},
				},
				{
					Type: "eval",
					Eval: &porchapi.FunctionEvalTaskSpec{
						// This function is expect to fail due to not knowing schema for some CRDs.
						Image: "gcr.io/kpt-fn/kubeval:v0.2.0",
					},
				},
			},
		},
	}
	err := t.client.Create(ctx, pr)
	expectedErrMsg := "Validating arbitrary CRDs is not supported"
	if err == nil || !strings.Contains(err.Error(), expectedErrMsg) {
		t.Fatalf("expected the error to contain %q, but got %v", expectedErrMsg, err)
	}
}

func (t *PorchSuite) TestRepositoryError(ctx context.Context) {
	const (
		repositoryName = "repo-with-error"
	)
	t.CreateF(ctx, &configapi.Repository{
		TypeMeta: metav1.TypeMeta{
			Kind:       configapi.RepositoryGVK.Kind,
			APIVersion: configapi.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      repositoryName,
			Namespace: t.namespace,
		},
		Spec: configapi.RepositorySpec{
			Description: "Repository With Error",
			Type:        configapi.RepositoryTypeGit,
			Content:     configapi.RepositoryContentPackage,
			Git: &configapi.GitRepository{
				// Use `incalid` domain: https://www.rfc-editor.org/rfc/rfc6761#section-6.4
				Repo: "https://repo.invalid/repository.git",
			},
		},
	})
	t.Cleanup(func() {
		t.DeleteL(ctx, &configapi.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      repositoryName,
				Namespace: t.namespace,
			},
		})
	})

	giveUp := time.Now().Add(60 * time.Second)

	for {
		if time.Now().After(giveUp) {
			t.Errorf("Timed out waiting for Repository Condition")
			break
		}

		time.Sleep(5 * time.Second)

		var repository configapi.Repository
		t.GetF(ctx, client.ObjectKey{
			Namespace: t.namespace,
			Name:      repositoryName,
		}, &repository)

		available := meta.FindStatusCondition(repository.Status.Conditions, configapi.RepositoryReady)
		if available == nil {
			// Condition not yet set
			t.Logf("Repository condition not yet available")
			continue
		}

		if got, want := available.Status, metav1.ConditionFalse; got != want {
			t.Errorf("Repository Available Condition Status; got %q, want %q", got, want)
		}
		if got, want := available.Reason, configapi.ReasonError; got != want {
			t.Errorf("Repository Available Condition Reason: got %q, want %q", got, want)
		}
		break
	}
}

func (t *PorchSuite) registerGitRepositoryF(ctx context.Context, repo, name, directory string) {
	t.CreateF(ctx, &configapi.Repository{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Repository",
			APIVersion: configapi.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: t.namespace,
		},
		Spec: configapi.RepositorySpec{
			Type:    configapi.RepositoryTypeGit,
			Content: configapi.RepositoryContentPackage,
			Git: &configapi.GitRepository{
				Repo:      repo,
				Branch:    "main",
				Directory: directory,
			},
		},
	})

	t.Cleanup(func() {
		t.DeleteL(ctx, &configapi.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: t.namespace,
			},
		})
	})
}

type repositoryOption func(*configapi.Repository)

func (t *PorchSuite) registerMainGitRepositoryF(ctx context.Context, name string, opts ...repositoryOption) {
	config := t.config

	var secret string
	// Create auth secret if necessary
	if config.Username != "" || config.Password != "" {
		secret = fmt.Sprintf("%s-auth", name)
		immutable := true
		t.CreateF(ctx, &coreapi.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret,
				Namespace: t.namespace,
			},
			Immutable: &immutable,
			Data: map[string][]byte{
				"username": []byte(config.Username),
				"password": []byte(config.Password),
			},
			Type: coreapi.SecretTypeBasicAuth,
		})

		t.Cleanup(func() {
			t.DeleteE(ctx, &coreapi.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secret,
					Namespace: t.namespace,
				},
			})
		})
	}

	repository := &configapi.Repository{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Repository",
			APIVersion: configapi.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: t.namespace,
		},
		Spec: configapi.RepositorySpec{
			Description: "Porch Test Repository Description",
			Type:        configapi.RepositoryTypeGit,
			Content:     configapi.RepositoryContentPackage,
			Git: &configapi.GitRepository{
				Repo:      config.Repo,
				Branch:    config.Branch,
				Directory: config.Directory,
				SecretRef: configapi.SecretRef{
					Name: secret,
				},
			},
		},
	}

	// Apply options
	for _, o := range opts {
		o(repository)
	}

	// Register repository
	t.CreateF(ctx, repository)

	t.Cleanup(func() {
		t.DeleteE(ctx, &configapi.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: t.namespace,
			},
		})
	})
}

func withDeployment() repositoryOption {
	return func(r *configapi.Repository) {
		r.Spec.Deployment = true
	}
}

func withType(t configapi.RepositoryType) repositoryOption {
	return func(r *configapi.Repository) {
		r.Spec.Type = t
	}
}

func withContent(content configapi.RepositoryContent) repositoryOption {
	return func(r *configapi.Repository) {
		r.Spec.Content = content
	}
}

// Creates an empty package draft by initializing an empty package
func (t *PorchSuite) createPackageDraftF(ctx context.Context, repository, name, revision string) *porchapi.PackageRevision {
	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    name,
			Revision:       revision,
			RepositoryName: repository,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeInit,
					Init: &porchapi.PackageInitTaskSpec{},
				},
			},
		},
	}
	t.CreateF(ctx, pr)
	return pr
}

func (t *PorchSuite) mustExist(ctx context.Context, key client.ObjectKey, obj client.Object) {
	t.GetF(ctx, key, obj)
	if got, want := obj.GetName(), key.Name; got != want {
		t.Errorf("%T.Name: got %q, want %q", obj, got, want)
	}
	if got, want := obj.GetNamespace(), key.Namespace; got != want {
		t.Errorf("%T.Namespace: got %q, want %q", obj, got, want)
	}
}

func (t *PorchSuite) mustNotExist(ctx context.Context, obj client.Object) {
	switch err := t.client.Get(ctx, client.ObjectKeyFromObject(obj), obj); {
	case err == nil:
		t.Errorf("No error returned getting a deleted package; expected error")
	case !apierrors.IsNotFound(err):
		t.Errorf("Expected NotFound error. got %v", err)
	}
}
