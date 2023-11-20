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
	internalapi "github.com/GoogleContainerTools/kpt/porch/internal/api/porchinternal/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	coreapi "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	testBlueprintsRepo = "https://github.com/platkrm/test-blueprints.git"
	kptRepo            = "https://github.com/GoogleContainerTools/kpt.git"
)

var (
	packageRevisionGVK = porchapi.SchemeGroupVersion.WithKind("PackageRevision")
	configMapGVK       = corev1.SchemeGroupVersion.WithKind("ConfigMap")
)

func TestE2E(t *testing.T) {
	e2e := os.Getenv("E2E")
	if e2e == "" {
		t.Skip("set E2E to run this test")
	}

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

	gitConfig GitConfig
}

var _ Initializer = &PorchSuite{}

func (p *PorchSuite) Initialize(ctx context.Context) {
	p.TestSuite.Initialize(ctx)
	p.gitConfig = p.CreateGitRepo()
}

func (p *PorchSuite) GitConfig(repoID string) GitConfig {
	config := p.gitConfig
	config.Repo = config.Repo + "/" + repoID
	return config
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
			WorkspaceName:  "workspace",
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

func (t *PorchSuite) TestGitRepositoryWithReleaseTagsAndDirectory(ctx context.Context) {
	t.registerGitRepositoryF(ctx, kptRepo, "kpt-repo", "package-examples")

	var list porchapi.PackageRevisionList
	t.ListF(ctx, &list, client.InNamespace(t.namespace))

	for _, pr := range list.Items {
		if strings.HasPrefix(pr.Spec.PackageName, "package-examples") {
			t.Errorf("package name %q should not include repo directory %q as prefix", pr.Spec.PackageName, "package-examples")
		}
	}
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
			WorkspaceName:  "test-workspace",
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
		workspace   = "test-workspace"
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
			WorkspaceName:  workspace,
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
		workspace   = "test-workspace"
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
			WorkspaceName:  workspace,
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
	const downstreamWorkspace = "test-workspace"

	// Register the deployment repository
	t.registerMainGitRepositoryF(ctx, downstreamRepository, withDeployment())

	// Register the upstream repository
	t.registerGitRepositoryF(ctx, testBlueprintsRepo, "test-blueprints", "")

	var upstreamPackages porchapi.PackageRevisionList
	t.ListE(ctx, &upstreamPackages, client.InNamespace(t.namespace))
	upstreamPackage := MustFindPackageRevision(t.T, &upstreamPackages, repository.PackageRevisionKey{
		Repository:    "test-blueprints",
		Package:       "basens",
		Revision:      "v1",
		WorkspaceName: "v1",
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
			WorkspaceName:  downstreamWorkspace,
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

func (t *PorchSuite) TestEditPackageRevision(ctx context.Context) {
	const (
		repository       = "edit-test"
		packageName      = "simple-package"
		otherPackageName = "other-package"
		workspace        = "workspace"
		workspace2       = "workspace2"
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
			WorkspaceName:  workspace,
			RepositoryName: repository,
		},
	}
	t.CreateF(ctx, pr)

	// Create a new revision, but with a different package as the source.
	// This is not allowed.
	invalidEditPR := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    otherPackageName,
			WorkspaceName:  workspace2,
			RepositoryName: repository,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeEdit,
					Edit: &porchapi.PackageEditTaskSpec{
						Source: &porchapi.PackageRevisionRef{
							Name: pr.Name,
						},
					},
				},
			},
		},
	}
	if err := t.client.Create(ctx, invalidEditPR); err == nil {
		t.Fatalf("Expected error for source revision being from different package")
	}

	// Create a new revision of the package with a source that is a revision
	// of the same package.
	editPR := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    packageName,
			WorkspaceName:  workspace2,
			RepositoryName: repository,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeEdit,
					Edit: &porchapi.PackageEditTaskSpec{
						Source: &porchapi.PackageRevisionRef{
							Name: pr.Name,
						},
					},
				},
			},
		},
	}
	if err := t.client.Create(ctx, editPR); err == nil {
		t.Fatalf("Expected error for source revision not being published")
	}

	// Publish the source package to make it a valid source for edit.
	pr.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, pr)

	// Approve the package
	pr.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	t.UpdateApprovalF(ctx, pr, metav1.UpdateOptions{})

	// Create a new revision with the edit task.
	t.CreateF(ctx, editPR)

	// Check its task list
	var pkgRev porchapi.PackageRevision
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      editPR.Name,
	}, &pkgRev)
	tasks := pkgRev.Spec.Tasks
	for _, tsk := range tasks {
		t.Logf("Task: %s", tsk.Type)
	}
	assert.Equal(t, 2, len(tasks))
}

// Test will initialize an empty package, update its resources, adding a function
// to the Kptfile's pipeline, and then check that the package was re-rendered.
func (t *PorchSuite) TestUpdateResources(ctx context.Context) {
	const (
		repository  = "re-render-test"
		packageName = "simple-package"
		workspace   = "workspace"
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
			WorkspaceName:  workspace,
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

	renderStatus := newPackage.Status.RenderStatus
	assert.Empty(t, renderStatus.Err, "render error must be empty for successful render operation.")
	assert.Zero(t, renderStatus.Result.ExitCode, "exit code must be zero for successful render operation.")
	assert.True(t, len(renderStatus.Result.Items) > 0)

	golden := filepath.Join("testdata", "update-resources", "want-config-map.yaml")
	if diff := t.CompareGoldenFileYAML(golden, updated); diff != "" {
		t.Errorf("Unexpected updated confg map contents: (-want,+got): %s", diff)
	}
}

// Test will initialize an empty package, and then make a call to update the resources
// without actually making any changes. This test is ensuring that no additional
// tasks get added.
func (t *PorchSuite) TestUpdateResourcesEmptyPatch(ctx context.Context) {
	const (
		repository  = "empty-patch-test"
		packageName = "simple-package"
		workspace   = "workspace"
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
			WorkspaceName:  workspace,
			RepositoryName: repository,
		},
	}
	t.CreateF(ctx, pr)

	// Check its task list
	var newPackage porchapi.PackageRevision
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &newPackage)
	tasksBeforeUpdate := newPackage.Spec.Tasks
	assert.Equal(t, 2, len(tasksBeforeUpdate))

	// Get the package resources
	var newPackageResources porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &newPackageResources)

	// "Update" the package resources, without changing anything
	t.UpdateF(ctx, &newPackageResources)

	// Check the task list
	var newPackageUpdated porchapi.PackageRevision
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &newPackageUpdated)
	tasksAfterUpdate := newPackageUpdated.Spec.Tasks
	assert.Equal(t, 2, len(tasksAfterUpdate))

	assert.True(t, reflect.DeepEqual(tasksBeforeUpdate, tasksAfterUpdate))
}

func (t *PorchSuite) TestFunctionRepository(ctx context.Context) {
	repo := &configapi.Repository{
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
	}
	t.CreateF(ctx, repo)

	t.Cleanup(func() {
		t.DeleteL(ctx, &configapi.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      repo.Name,
				Namespace: repo.Namespace,
			},
		})
	})

	// Make sure the repository is ready before we test to (hopefully)
	// avoid flakiness.
	t.waitUntilRepositoryReady(ctx, repo.Name, repo.Namespace)

	// Wait here for the repository to be cached in porch. We wait
	// first one minute, since Porch waits 1 minute before it syncs
	// the repo for the first time. Then wait another minute so that
	// the sync has (hopefully) finished.
	// TODO(mortent): We need a better solution for this. This is only
	// temporary to fix the current flakiness with the e2e tests.
	<-time.NewTimer(2 * time.Minute).C

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
		repository  = "lifecycle"
		packageName = "test-package"
		workspace   = "workspace"
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
			WorkspaceName:  workspace,
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

	// Check its revision number
	if got, want := approved.Spec.Revision, "v1"; got != want {
		t.Fatalf("Approved package revision value: got %s, want %s", got, want)
	}
}

func (t *PorchSuite) TestDeleteDraft(ctx context.Context) {
	const (
		repository  = "delete-draft"
		packageName = "test-delete-draft"
		revision    = "v1"
		workspace   = "test-workspace"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	created := t.createPackageDraftF(ctx, repository, packageName, workspace)

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
		workspace   = "workspace"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	created := t.createPackageDraftF(ctx, repository, packageName, workspace)

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
		workspace   = "workspace"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	created := t.createPackageDraftF(ctx, repository, packageName, workspace)

	// Check the package exists
	var pkg porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Propose the package revision to be finalized
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkg)

	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	t.UpdateApprovalF(ctx, &pkg, metav1.UpdateOptions{})

	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Try to delete the package. This should fail because it hasn't been proposed for deletion.
	t.DeleteL(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      created.Name,
		},
	})
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Propose deletion and then delete the package
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleDeletionProposed
	t.UpdateApprovalF(ctx, &pkg, metav1.UpdateOptions{})

	t.DeleteE(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      created.Name,
		},
	})

	t.mustNotExist(ctx, &pkg)
}

func (t *PorchSuite) TestProposeDeleteAndUndo(ctx context.Context) {
	const (
		repository  = "test-propose-delete-and-undo"
		packageName = "test-propose-delete-and-undo"
		workspace   = "workspace"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	created := t.createPackageDraftF(ctx, repository, packageName, workspace)

	// Check the package exists
	var pkg porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Propose the package revision to be finalized
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkg)

	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	t.UpdateApprovalF(ctx, &pkg, metav1.UpdateOptions{})
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	t.waitUntilMainBranchPackageRevisionExists(ctx, packageName)

	var list porchapi.PackageRevisionList
	t.ListF(ctx, &list, client.InNamespace(t.namespace))

	for i := range list.Items {
		pkgRev := list.Items[i]
		t.Run(fmt.Sprintf("revision %s", pkgRev.Spec.Revision), func(newT *testing.T) {
			// This is a bit awkward, we should find a better way to allow subtests
			// with our custom implmentation of t.
			oldT := t.T
			t.T = newT
			defer func() {
				t.T = oldT
			}()

			// Propose deletion
			pkgRev.Spec.Lifecycle = porchapi.PackageRevisionLifecycleDeletionProposed
			t.UpdateApprovalF(ctx, &pkgRev, metav1.UpdateOptions{})

			// Undo proposal of deletion
			pkgRev.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
			t.UpdateApprovalF(ctx, &pkgRev, metav1.UpdateOptions{})

			// Try to delete the package. This should fail because the lifecycle should be changed back to Published.
			t.DeleteL(ctx, &porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: t.namespace,
					Name:      pkgRev.Name,
				},
			})
			t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: pkgRev.Name}, &pkgRev)

			// Propose deletion and then delete the package
			pkgRev.Spec.Lifecycle = porchapi.PackageRevisionLifecycleDeletionProposed
			t.UpdateApprovalF(ctx, &pkgRev, metav1.UpdateOptions{})

			t.DeleteE(ctx, &porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: t.namespace,
					Name:      pkgRev.Name,
				},
			})

			t.mustNotExist(ctx, &pkgRev)
		})
	}
}

func (t *PorchSuite) TestDeleteAndRecreate(ctx context.Context) {
	const (
		repository  = "delete-and-recreate"
		packageName = "test-delete-and-recreate"
		revision    = "v1"
		workspace   = "work"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	created := t.createPackageDraftF(ctx, repository, packageName, workspace)

	// Check the package exists
	var pkg porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Propose the package revision to be finalized
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkg)

	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	t.UpdateApprovalF(ctx, &pkg, metav1.UpdateOptions{})

	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: created.Name}, &pkg)

	// Propose deletion and then delete the package
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleDeletionProposed
	t.UpdateApprovalF(ctx, &pkg, metav1.UpdateOptions{})

	t.DeleteE(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      created.Name,
		},
	})

	t.mustNotExist(ctx, &pkg)

	// Recreate the package with the same name and workspace
	created = t.createPackageDraftF(ctx, repository, packageName, workspace)

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

func (t *PorchSuite) TestDeleteFromMain(ctx context.Context) {
	const (
		repository        = "delete-main"
		packageNameFirst  = "test-delete-main-first"
		packageNameSecond = "test-delete-main-second"
		workspace         = "workspace"
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create the first draft package
	createdFirst := t.createPackageDraftF(ctx, repository, packageNameFirst, workspace)

	// Check the package exists
	var pkgFirst porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: createdFirst.Name}, &pkgFirst)

	// Propose the package revision to be finalized
	pkgFirst.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkgFirst)

	pkgFirst.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	t.UpdateApprovalF(ctx, &pkgFirst, metav1.UpdateOptions{})

	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: createdFirst.Name}, &pkgFirst)

	// Create the second draft package
	createdSecond := t.createPackageDraftF(ctx, repository, packageNameSecond, workspace)

	// Check the package exists
	var pkgSecond porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: createdSecond.Name}, &pkgSecond)

	// Propose the package revision to be finalized
	pkgSecond.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkgSecond)

	pkgSecond.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	t.UpdateApprovalF(ctx, &pkgSecond, metav1.UpdateOptions{})

	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: createdSecond.Name}, &pkgSecond)

	// We need to wait for the sync for the "main" revisions to get created
	time.Sleep(75 * time.Second)

	var list porchapi.PackageRevisionList
	t.ListE(ctx, &list, client.InNamespace(t.namespace))

	var firstPkgRevFromMain porchapi.PackageRevision
	var secondPkgRevFromMain porchapi.PackageRevision

	for _, pkgrev := range list.Items {
		if pkgrev.Spec.PackageName == packageNameFirst && pkgrev.Spec.Revision == "main" {
			firstPkgRevFromMain = pkgrev
		}
		if pkgrev.Spec.PackageName == packageNameSecond && pkgrev.Spec.Revision == "main" {
			secondPkgRevFromMain = pkgrev
		}
	}

	// Propose deletion of both main packages
	firstPkgRevFromMain.Spec.Lifecycle = porchapi.PackageRevisionLifecycleDeletionProposed
	t.UpdateApprovalF(ctx, &firstPkgRevFromMain, metav1.UpdateOptions{})
	secondPkgRevFromMain.Spec.Lifecycle = porchapi.PackageRevisionLifecycleDeletionProposed
	t.UpdateApprovalF(ctx, &secondPkgRevFromMain, metav1.UpdateOptions{})

	// Delete the first package revision from main
	t.DeleteE(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      firstPkgRevFromMain.Name,
		},
	})

	// We need to wait for the sync
	time.Sleep(75 * time.Second)

	// Delete the second package revision from main
	t.DeleteE(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      secondPkgRevFromMain.Name,
		},
	})

	// Propose and delete the original package revisions (cleanup)
	t.ListE(ctx, &list, client.InNamespace(t.namespace))
	for _, pkgrev := range list.Items {
		pkgrev.Spec.Lifecycle = porchapi.PackageRevisionLifecycleDeletionProposed
		t.UpdateApprovalF(ctx, &pkgrev, metav1.UpdateOptions{})
		t.DeleteE(ctx, &porchapi.PackageRevision{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: t.namespace,
				Name:      pkgrev.Name,
			},
		})
	}
}

func (t *PorchSuite) TestCloneLeadingSlash(ctx context.Context) {
	const (
		repository  = "clone-ls"
		packageName = "test-clone-ls"
		revision    = "v1"
		workspace   = "workspace"
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
			WorkspaceName:  workspace,
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
			WorkspaceName:  "test-workspace",
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

	var newrr porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      pr.Name,
	}, &newrr)

	by, _ := yaml.Marshal(&newrr)
	t.Logf("PRR: %s", string(by))

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
			WorkspaceName:  "test-workspace",
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
			WorkspaceName:  "test-workspace",
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
			WorkspaceName:  "test-description",
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
			WorkspaceName:  "workspace-1",
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
			WorkspaceName:  "workspace-2",
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
			WorkspaceName:  "workspace",
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
			Kind:       configapi.TypeRepository.Kind,
			APIVersion: configapi.TypeRepository.APIVersion(),
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

func (t *PorchSuite) TestNewPackageRevisionLabels(ctx context.Context) {
	const (
		repository = "pkg-rev-labels"
		labelKey1  = "kpt.dev/label"
		labelVal1  = "foo"
		labelKey2  = "kpt.dev/other-label"
		labelVal2  = "bar"
		annoKey1   = "kpt.dev/anno"
		annoVal1   = "foo"
		annoKey2   = "kpt.dev/other-anno"
		annoVal2   = "bar"
	)

	t.registerMainGitRepositoryF(ctx, repository)

	// Create a package with labels and annotations.
	pr := porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Labels: map[string]string{
				labelKey1: labelVal1,
			},
			Annotations: map[string]string{
				annoKey1: annoVal1,
				annoKey2: annoVal2,
			},
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "new-package",
			WorkspaceName:  "workspace",
			RepositoryName: repository,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeInit,
					Init: &porchapi.PackageInitTaskSpec{
						Description: "this is a test",
					},
				},
			},
		},
	}
	t.CreateF(ctx, &pr)
	t.validateLabelsAndAnnos(ctx, pr.Name,
		map[string]string{
			labelKey1: labelVal1,
		},
		map[string]string{
			annoKey1: annoVal1,
			annoKey2: annoVal2,
		},
	)

	// Propose the package.
	pr.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pr)

	// retrieve the updated object
	t.GetF(ctx, client.ObjectKey{
		Namespace: pr.Namespace,
		Name:      pr.Name,
	}, &pr)

	t.validateLabelsAndAnnos(ctx, pr.Name,
		map[string]string{
			labelKey1: labelVal1,
		},
		map[string]string{
			annoKey1: annoVal1,
			annoKey2: annoVal2,
		},
	)

	// Approve the package
	pr.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	_ = t.UpdateApprovalF(ctx, &pr, metav1.UpdateOptions{})
	t.validateLabelsAndAnnos(ctx, pr.Name,
		map[string]string{
			labelKey1:                         labelVal1,
			porchapi.LatestPackageRevisionKey: porchapi.LatestPackageRevisionValue,
		},
		map[string]string{
			annoKey1: annoVal1,
			annoKey2: annoVal2,
		},
	)

	// retrieve the updated object
	t.GetF(ctx, client.ObjectKey{
		Namespace: pr.Namespace,
		Name:      pr.Name,
	}, &pr)

	// Update the labels and annotations on the approved package.
	delete(pr.ObjectMeta.Labels, labelKey1)
	pr.ObjectMeta.Labels[labelKey2] = labelVal2
	delete(pr.ObjectMeta.Annotations, annoKey2)
	pr.Spec.Revision = "v1"
	t.UpdateF(ctx, &pr)
	t.validateLabelsAndAnnos(ctx, pr.Name,
		map[string]string{
			labelKey2:                         labelVal2,
			porchapi.LatestPackageRevisionKey: porchapi.LatestPackageRevisionValue,
		},
		map[string]string{
			annoKey1: annoVal1,
		},
	)

	// Create PackageRevision from upstream repo. Labels and annotations should
	// not be retained from upstream.
	clonedPr := porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "cloned-package",
			WorkspaceName:  "workspace",
			RepositoryName: repository,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeClone,
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							UpstreamRef: &porchapi.PackageRevisionRef{
								Name: pr.Name, // Package to be cloned
							},
						},
					},
				},
			},
		},
	}
	t.CreateF(ctx, &clonedPr)
	t.validateLabelsAndAnnos(ctx, clonedPr.Name,
		map[string]string{},
		map[string]string{},
	)
}

func (t *PorchSuite) TestRegisteredPackageRevisionLabels(ctx context.Context) {
	const (
		labelKey = "kpt.dev/label"
		labelVal = "foo"
		annoKey  = "kpt.dev/anno"
		annoVal  = "foo"
	)

	t.registerGitRepositoryF(ctx, testBlueprintsRepo, "test-blueprints", "")

	var list porchapi.PackageRevisionList
	t.ListE(ctx, &list, client.InNamespace(t.namespace))

	basens := MustFindPackageRevision(t.T, &list, repository.PackageRevisionKey{Repository: "test-blueprints", Package: "basens", Revision: "v1"})
	if basens.ObjectMeta.Labels == nil {
		basens.ObjectMeta.Labels = make(map[string]string)
	}
	basens.ObjectMeta.Labels[labelKey] = labelVal
	if basens.ObjectMeta.Annotations == nil {
		basens.ObjectMeta.Annotations = make(map[string]string)
	}
	basens.ObjectMeta.Annotations[annoKey] = annoVal
	t.UpdateF(ctx, basens)

	t.validateLabelsAndAnnos(ctx, basens.Name,
		map[string]string{
			labelKey: labelVal,
		},
		map[string]string{
			annoKey: annoVal,
		},
	)
}

func (t *PorchSuite) TestPackageRevisionGCWithOwner(ctx context.Context) {
	const (
		repository  = "pkgrevgcwithowner"
		workspace   = "pkgrevgcwithowner-workspace"
		description = "empty-package description"
		cmName      = "foo"
	)

	t.registerMainGitRepositoryF(ctx, repository)

	// Create a new package (via init)
	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       packageRevisionGVK.Kind,
			APIVersion: packageRevisionGVK.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "empty-package",
			WorkspaceName:  workspace,
			RepositoryName: repository,
		},
	}
	t.CreateF(ctx, pr)

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       configMapGVK.Kind,
			APIVersion: configMapGVK.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: t.namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: porchapi.SchemeGroupVersion.String(),
					Kind:       packageRevisionGVK.Kind,
					Name:       pr.Name,
					UID:        pr.UID,
				},
			},
		},
		Data: map[string]string{
			"foo": "bar",
		},
	}
	t.CreateF(ctx, cm)

	t.DeleteF(ctx, pr)
	t.waitUntilObjectDeleted(
		ctx,
		packageRevisionGVK,
		types.NamespacedName{
			Name:      pr.Name,
			Namespace: pr.Namespace,
		},
		10*time.Second,
	)
	t.waitUntilObjectDeleted(
		ctx,
		configMapGVK,
		types.NamespacedName{
			Name:      cm.Name,
			Namespace: cm.Namespace,
		},
		10*time.Second,
	)
}

func (t *PorchSuite) TestPackageRevisionGCAsOwner(ctx context.Context) {
	const (
		repository  = "pkgrevgcasowner"
		workspace   = "pkgrevgcasowner-workspace"
		description = "empty-package description"
		cmName      = "foo"
	)

	t.registerMainGitRepositoryF(ctx, repository)

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       configMapGVK.Kind,
			APIVersion: configMapGVK.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: t.namespace,
		},
		Data: map[string]string{
			"foo": "bar",
		},
	}
	t.CreateF(ctx, cm)

	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       packageRevisionGVK.Kind,
			APIVersion: packageRevisionGVK.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       cm.Name,
					UID:        cm.UID,
				},
			},
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "empty-package",
			WorkspaceName:  workspace,
			RepositoryName: repository,
		},
	}
	t.CreateF(ctx, pr)

	t.DeleteF(ctx, cm)
	t.waitUntilObjectDeleted(
		ctx,
		configMapGVK,
		types.NamespacedName{
			Name:      cm.Name,
			Namespace: cm.Namespace,
		},
		10*time.Second,
	)
	t.waitUntilObjectDeleted(
		ctx,
		packageRevisionGVK,
		types.NamespacedName{
			Name:      pr.Name,
			Namespace: pr.Namespace,
		},
		10*time.Second,
	)
}

func (t *PorchSuite) TestPackageRevisionOwnerReferences(ctx context.Context) {
	const (
		repository  = "pkgrevownerrefs"
		workspace   = "pkgrevownerrefs-workspace"
		description = "empty-package description"
		cmName      = "foo"
	)

	t.registerMainGitRepositoryF(ctx, repository)

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       configMapGVK.Kind,
			APIVersion: configMapGVK.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: t.namespace,
		},
		Data: map[string]string{
			"foo": "bar",
		},
	}
	t.CreateF(ctx, cm)

	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       packageRevisionGVK.Kind,
			APIVersion: packageRevisionGVK.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "empty-package",
			WorkspaceName:  workspace,
			RepositoryName: repository,
		},
	}
	t.CreateF(ctx, pr)
	t.validateOwnerReferences(ctx, pr.Name, []metav1.OwnerReference{})

	ownerRef := metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Name:       cm.Name,
		UID:        cm.UID,
	}
	pr.ObjectMeta.OwnerReferences = []metav1.OwnerReference{ownerRef}
	t.UpdateF(ctx, pr)
	t.validateOwnerReferences(ctx, pr.Name, []metav1.OwnerReference{ownerRef})

	pr.ObjectMeta.OwnerReferences = []metav1.OwnerReference{}
	t.UpdateF(ctx, pr)
	t.validateOwnerReferences(ctx, pr.Name, []metav1.OwnerReference{})
}

func (t *PorchSuite) TestPackageRevisionFinalizers(ctx context.Context) {
	const (
		repository  = "pkgrevfinalizers"
		workspace   = "pkgrevfinalizers-workspace"
		description = "empty-package description"
	)

	t.registerMainGitRepositoryF(ctx, repository)

	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       packageRevisionGVK.Kind,
			APIVersion: packageRevisionGVK.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "empty-package",
			WorkspaceName:  workspace,
			RepositoryName: repository,
		},
	}
	t.CreateF(ctx, pr)
	t.validateFinalizers(ctx, pr.Name, []string{})

	pr.Finalizers = append(pr.Finalizers, "foo-finalizer")
	t.UpdateF(ctx, pr)
	t.validateFinalizers(ctx, pr.Name, []string{"foo-finalizer"})

	t.DeleteF(ctx, pr)
	t.validateFinalizers(ctx, pr.Name, []string{"foo-finalizer"})

	pr.Finalizers = []string{}
	t.UpdateF(ctx, pr)
	t.waitUntilObjectDeleted(ctx, packageRevisionGVK, types.NamespacedName{
		Name:      pr.Name,
		Namespace: pr.Namespace,
	}, 10*time.Second)
}

func (t *PorchSuite) validateFinalizers(ctx context.Context, name string, finalizers []string) {
	var pr porchapi.PackageRevision
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      name,
	}, &pr)

	if len(finalizers) != len(pr.Finalizers) {
		diff := cmp.Diff(finalizers, pr.Finalizers)
		t.Errorf("Expected %d finalizers, but got %s", len(finalizers), diff)
	}

	for _, finalizer := range finalizers {
		var found bool
		for _, f := range pr.Finalizers {
			if f == finalizer {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected finalizer %v, but didn't find it", finalizer)
		}
	}
}

func (t *PorchSuite) validateOwnerReferences(ctx context.Context, name string, ownerRefs []metav1.OwnerReference) {
	var pr porchapi.PackageRevision
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      name,
	}, &pr)

	if len(ownerRefs) != len(pr.OwnerReferences) {
		diff := cmp.Diff(ownerRefs, pr.OwnerReferences)
		t.Errorf("Expected %d ownerReferences, but got %s", len(ownerRefs), diff)
	}

	for _, ownerRef := range ownerRefs {
		var found bool
		for _, or := range pr.OwnerReferences {
			if or == ownerRef {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected ownerRef %v, but didn't find it", ownerRef)
		}
	}
}

func (t *PorchSuite) validateLabelsAndAnnos(ctx context.Context, name string, labels, annos map[string]string) {
	var pr porchapi.PackageRevision
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      name,
	}, &pr)

	actualLabels := pr.ObjectMeta.Labels
	actualAnnos := pr.ObjectMeta.Annotations

	// Make this check to handle empty vs nil maps
	if !(len(labels) == 0 && len(actualLabels) == 0) {
		if diff := cmp.Diff(actualLabels, labels); diff != "" {
			t.Errorf("Unexpected result (-want, +got): %s", diff)
		}
	}

	if !(len(annos) == 0 && len(actualAnnos) == 0) {
		if diff := cmp.Diff(actualAnnos, annos); diff != "" {
			t.Errorf("Unexpected result (-want, +got): %s", diff)
		}
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

	// Make sure the repository is ready before we test to (hopefully)
	// avoid flakiness.
	t.waitUntilRepositoryReady(ctx, name, t.namespace)
}

type repositoryOption func(*configapi.Repository)

func (t *PorchSuite) registerMainGitRepositoryF(ctx context.Context, name string, opts ...repositoryOption) {
	repoID := t.namespace + "-" + name
	config := t.GitConfig(repoID)

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
		t.waitUntilRepositoryDeleted(ctx, name, t.namespace)
		t.waitUntilAllPackagesDeleted(ctx, name)
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
func (t *PorchSuite) createPackageDraftF(ctx context.Context, repository, name, workspace string) *porchapi.PackageRevision {
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
			WorkspaceName:  porchapi.WorkspaceName(workspace),
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

// waitUntilRepositoryReady waits for up to 10 seconds for the repository with the
// provided name and namespace is ready, i.e. the Ready condition is true.
// It also queries for Functions and PackageRevisions, to ensure these are also
// ready - this is an artifact of the way we've implemented the aggregated apiserver,
// where the first fetch will block on the cache loading. Wait up to two minutes for the
// package revisions and functions.
func (t *PorchSuite) waitUntilRepositoryReady(ctx context.Context, name, namespace string) {
	nn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	var innerErr error
	err := wait.PollImmediateWithContext(ctx, time.Second, 10*time.Second, func(ctx context.Context) (bool, error) {
		var repo configapi.Repository
		if err := t.client.Get(ctx, nn, &repo); err != nil {
			innerErr = err
			return false, nil
		}
		for _, c := range repo.Status.Conditions {
			if c.Type == configapi.RepositoryReady {
				return c.Status == metav1.ConditionTrue, nil
			}
		}
		return false, nil
	})
	if err != nil {
		t.Errorf("Repository not ready after wait: %v", innerErr)
	}

	// While we're using an aggregated apiserver, make sure we can query the generated objects
	if err := wait.PollImmediateWithContext(ctx, time.Second, 120*time.Second, func(ctx context.Context) (bool, error) {
		var revisions porchapi.PackageRevisionList
		if err := t.client.List(ctx, &revisions, client.InNamespace(nn.Namespace)); err != nil {
			innerErr = err
			return false, nil
		}
		return true, nil
	}); err != nil {
		t.Errorf("unable to query PackageRevisions after wait: %v", innerErr)
	}

	// Check for functions also (until we move them to CRDs)
	if err := wait.PollImmediateWithContext(ctx, time.Second, 120*time.Second, func(ctx context.Context) (bool, error) {
		var functions porchapi.FunctionList
		if err := t.client.List(ctx, &functions, client.InNamespace(nn.Namespace)); err != nil {
			innerErr = err
			return false, nil
		}
		return true, nil
	}); err != nil {
		t.Errorf("unable to query Functions after wait: %v", innerErr)
	}

}

func (t *PorchSuite) waitUntilRepositoryDeleted(ctx context.Context, name, namespace string) {
	err := wait.PollImmediateWithContext(ctx, time.Second, 20*time.Second, func(ctx context.Context) (done bool, err error) {
		var repo configapi.Repository
		nn := types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}
		if err := t.client.Get(ctx, nn, &repo); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("Repository %s/%s not deleted", namespace, name)
	}
}

func (t *PorchSuite) waitUntilAllPackagesDeleted(ctx context.Context, repoName string) {
	err := wait.PollImmediateWithContext(ctx, time.Second, 60*time.Second, func(ctx context.Context) (done bool, err error) {
		var pkgRevList porchapi.PackageRevisionList
		if err := t.client.List(ctx, &pkgRevList); err != nil {
			t.Logf("error listing packages: %v", err)
			return false, nil
		}
		for _, pkgRev := range pkgRevList.Items {
			if strings.HasPrefix(fmt.Sprintf("%s-", pkgRev.Name), repoName) {
				t.Logf("Found package %s from repo %s", pkgRev.Name, repoName)
				return false, nil
			}
		}

		var internalPkgRevList internalapi.PackageRevList
		if err := t.client.List(ctx, &internalPkgRevList); err != nil {
			t.Logf("error list internal packages: %v", err)
			return false, nil
		}
		for _, internalPkgRev := range internalPkgRevList.Items {
			if strings.HasPrefix(fmt.Sprintf("%s-", internalPkgRev.Name), repoName) {
				t.Logf("Found internalPkg %s from repo %s", internalPkgRev.Name, repoName)
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Packages from repo %s still remains", repoName)
	}
}

func (t *PorchSuite) waitUntilObjectDeleted(ctx context.Context, gvk schema.GroupVersionKind, namespacedName types.NamespacedName, d time.Duration) {
	var innerErr error
	err := wait.PollImmediateWithContext(ctx, time.Second, d, func(ctx context.Context) (bool, error) {
		var u unstructured.Unstructured
		u.SetGroupVersionKind(gvk)
		if err := t.client.Get(ctx, namespacedName, &u); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			innerErr = err
			return false, err
		}
		return false, nil
	})
	if err != nil {
		t.Errorf("Object %s not deleted after %s: %v", namespacedName.String(), d.String(), innerErr)
	}
}

func (t *PorchSuite) waitUntilMainBranchPackageRevisionExists(ctx context.Context, pkgName string) {
	err := wait.PollImmediateWithContext(ctx, time.Second, 120*time.Second, func(ctx context.Context) (done bool, err error) {
		var pkgRevList porchapi.PackageRevisionList
		if err := t.client.List(ctx, &pkgRevList); err != nil {
			t.Logf("error listing packages: %v", err)
			return false, nil
		}
		for _, pkgRev := range pkgRevList.Items {
			pkgName := pkgRev.Spec.PackageName
			pkgRevision := pkgRev.Spec.Revision
			if pkgRevision == "main" &&
				pkgName == pkgRev.Spec.PackageName {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("Main branch package revision for %s not found", pkgName)
	}
}
