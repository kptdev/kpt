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
	"reflect"
	"strings"
	"testing"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/google/go-cmp/cmp"
	coreapi "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	testBlueprintsRepo = "https://github.com/platkrm/test-blueprints.git"
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
	t.CreateF(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git:test-bucket:v1",
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
						Image: "gcr.io/kpt-fn/set-namespace:unstable",
						ConfigMap: map[string]string{
							"namespace": "bucket-namespace",
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
	})

	// Get package resources
	var resources porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      "git:test-bucket:v1",
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

func (t *PorchSuite) TestCloneFromUpstream(ctx context.Context) {
	// Register Upstream Repository
	t.registerGitRepositoryF(ctx, testBlueprintsRepo, "test-blueprints")

	var pr porchapi.PackageRevisionResourcesList
	t.ListE(ctx, &pr, client.InNamespace(t.namespace))

	// Ensure basens package exists
	const name = "test-blueprints:basens:v1"
	found := false
	for _, r := range pr.Items {
		if r.Name == name {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Repository %q doesn't contain package %q", testBlueprintsRepo, name)
	}

	// Register the repository as 'downstream'
	t.registerMainGitRepositoryF(ctx, "downstream")

	// Create PackageRevision from upstream repo
	t.CreateF(ctx, &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "downstream:istions:v1",
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
							UpstreamRef: porchapi.PackageRevisionRef{
								Name: "test-blueprints:basens:v1", // Clone from basens/v1
							},
						},
					},
				},
			},
		},
	})

	// Get istions resources
	var istions porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      "downstream:istions:v1",
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
			Ref:       "v1",
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
			Ref:       "v1",
		},
	}); !cmp.Equal(want, got) {
		t.Errorf("unexpected upstream returned (-want, +got) %s", cmp.Diff(want, got))
	}
}

func (t *PorchSuite) TestInitEmptyPackage(ctx context.Context) {
	// Create a new package via init, no task specified
	const repository = "git"
	const name = repository + ":empty-package:v1"
	const description = "empty-package description"

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a new package (via init)
	t.CreateF(ctx, &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: t.namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    "empty-package",
			Revision:       "v1",
			RepositoryName: repository,
		},
	})

	// Get the package
	var newPackage porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      name,
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
	const repository = "git"
	const name = repository + ":new-package:v1"
	const description = "New Package"
	const site = "https://kpt.dev/new-package"
	keywords := []string{"test"}

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a new package (via init)
	t.CreateF(ctx, &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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
	})

	// Get the package
	var newPackage porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      name,
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
	const downstreamRevision = "v1"
	const downstreamName = downstreamRepository + ":" + downstreamPackage + ":" + downstreamRevision

	// Register the deployment repository
	t.registerMainGitRepositoryF(ctx, downstreamRepository, withDeployment())

	// Register the upstream repository
	t.registerGitRepositoryF(ctx, testBlueprintsRepo, "test-blueprints")

	// TODO: Confirm that upstream package doesn't contain context

	const upstreamPackage = "test-blueprints:basens:v1"

	// Create PackageRevision from upstream repo
	t.CreateF(ctx, &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      downstreamName,
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
							UpstreamRef: porchapi.PackageRevisionRef{
								Name: upstreamPackage, // Package to be cloned
							},
						},
					},
				},
			},
		},
	})

	// Get istions resources
	var istions porchapi.PackageRevisionResources
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      downstreamName,
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
			Ref:       "v1",
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
			Ref:       "v1",
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

func (t *PorchSuite) TestFunctionRepository(ctx context.Context) {
	t.CreateF(ctx, &configapi.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "function-repository",
			Namespace: t.namespace,
		},
		Spec: configapi.RepositorySpec{
			Title:       "Function Repository",
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
	t.registerGitRepositoryF(ctx, testBlueprintsRepo, "demo-blueprints")

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
		fullName        = repository + ":" + packageName + ":" + packageRevision
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a new package (via init)
	t.CreateF(ctx, &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fullName,
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
	})

	var pkg porchapi.PackageRevision
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      fullName,
	}, &pkg)

	// Propose the package revision to be finalized
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkg)

	var proposed porchapi.PackageRevision
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      fullName,
	}, &proposed)

	if got, want := proposed.Spec.Lifecycle, porchapi.PackageRevisionLifecycleProposed; got != want {
		t.Fatalf("Proposed package lifecycle value: got %s, want %s", got, want)
	}

	// Approve using Update should fail.
	proposed.Spec.Lifecycle = porchapi.PackageRevisionLifecycleFinal
	if err := t.client.Update(ctx, &proposed); err == nil {
		t.Fatalf("Finalization of a package via Update unexpectedly succeeded")
	}

	// Approve the package
	proposed.Spec.Lifecycle = porchapi.PackageRevisionLifecycleFinal
	approved := t.UpdateApprovalF(ctx, &proposed, metav1.UpdateOptions{})
	if got, want := approved.Spec.Lifecycle, porchapi.PackageRevisionLifecycleFinal; got != want {
		t.Fatalf("Approved package lifecycle value: got %s, want %s", got, want)
	}
}

func (t *PorchSuite) TestDeleteDraft(ctx context.Context) {
	const (
		repository  = "delete-draft"
		packageName = "test-delete-draft"
		revision    = "v1"
		name        = repository + ":" + packageName + ":" + revision
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	t.createPackageDraftF(ctx, repository, packageName, revision)

	// Check the package exists
	var draft porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: name}, &draft)

	// Delete the package
	t.DeleteE(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      name,
		},
	})

	t.mustNotExist(ctx, &draft)
}

func (t *PorchSuite) TestDeleteProposed(ctx context.Context) {
	const (
		repository  = "delete-proposed"
		packageName = "test-delete-proposed"
		revision    = "v1"
		name        = repository + ":" + packageName + ":" + revision
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	t.createPackageDraftF(ctx, repository, packageName, revision)

	// Check the package exists
	var pkg porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: name}, &pkg)

	// Propose the package revision to be finalized
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkg)

	// Delete the package
	t.DeleteE(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      name,
		},
	})

	t.mustNotExist(ctx, &pkg)
}

func (t *PorchSuite) TestDeleteFinal(ctx context.Context) {
	const (
		repository  = "delete-final"
		packageName = "test-delete-final"
		revision    = "v1"
		name        = repository + ":" + packageName + ":" + revision
	)

	// Register the repository
	t.registerMainGitRepositoryF(ctx, repository)

	// Create a draft package
	t.createPackageDraftF(ctx, repository, packageName, revision)

	// Check the package exists
	var pkg porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: name}, &pkg)

	// Propose the package revision to be finalized
	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	t.UpdateF(ctx, &pkg)

	pkg.Spec.Lifecycle = porchapi.PackageRevisionLifecycleFinal
	t.UpdateApprovalF(ctx, &pkg, metav1.UpdateOptions{})

	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: name}, &pkg)

	// Delete the package
	t.DeleteE(ctx, &porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.namespace,
			Name:      name,
		},
	})

	t.mustNotExist(ctx, &pkg)
}

func (t *PorchSuite) TestCloneLeadingSlash(ctx context.Context) {
	const (
		repository  = "clone-ls"
		packageName = "test-clone-ls"
		revision    = "v1"
		name        = repository + ":" + packageName + ":" + revision
	)

	t.registerMainGitRepositoryF(ctx, repository)

	// Clone the package. Use leading slash in the directory (regression test)
	t.CreateF(ctx, &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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
	})

	var pr porchapi.PackageRevision
	t.mustExist(ctx, client.ObjectKey{Namespace: t.namespace, Name: name}, &pr)
}

func (t *PorchSuite) TestRegisterRepository(ctx context.Context) {
	const (
		repository = "register"
		title      = "Test Register Repository"
	)
	t.registerMainGitRepositoryF(ctx, repository,
		withTitle(title),
		withContent(configapi.RepositoryContentPackage),
		withType(configapi.RepositoryTypeGit),
		withDeployment())

	var repo configapi.Repository
	t.GetF(ctx, client.ObjectKey{
		Namespace: t.namespace,
		Name:      repository,
	}, &repo)

	if got, want := repo.Spec.Title, title; got != want {
		t.Errorf("Repo Title: got %q, want %q", got, want)
	}
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

func (t *PorchSuite) registerGitRepositoryF(ctx context.Context, repo, name string) {
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
			Title:   "Public Git Repository",
			Type:    configapi.RepositoryTypeGit,
			Content: configapi.RepositoryContentPackage,
			Git: &configapi.GitRepository{
				Repo:   repo,
				Branch: "main",
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
			Title:       "Porch Test Repository",
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

func withTitle(title string) repositoryOption {
	return func(r *configapi.Repository) {
		r.Spec.Title = title
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
	fullName := fmt.Sprintf("%s:%s:%s", repository, name, revision)
	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fullName,
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
