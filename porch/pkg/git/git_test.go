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

package git

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func TestMain(m *testing.M) {
	klog.InitFlags(nil)
	flag.Parse()
	os.Exit(m.Run())
}

func TestGit(t *testing.T) {
	for _, gs := range []GitSuite{
		{branch: "main"},
		{branch: "feature"},
		{branch: "nested/release"},
	} {
		name := strings.ReplaceAll(gs.branch, "/", "-")
		t.Run(name, func(t *testing.T) {
			Run(gs, t)
		})
	}
}

func Run(suite interface{}, t *testing.T) {
	sv := reflect.ValueOf(suite)
	st := reflect.TypeOf(suite)

	for i, max := 0, st.NumMethod(); i < max; i++ {
		m := st.Method(i)
		if strings.HasPrefix(m.Name, "Test") {
			t.Run(m.Name, func(t *testing.T) {
				m.Func.Call([]reflect.Value{sv, reflect.ValueOf((t))})
			})
		}
	}
}

type GitSuite struct {
	branch string
}

func (g GitSuite) TestOpenEmptyRepository(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "empty-repository.tar")
	repo, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

	ctx := context.Background()
	const (
		name      = "empty"
		namespace = "default"
	)

	repository := &configapi.GitRepository{
		Repo:      address,
		Branch:    g.branch,
		Directory: "/",
	}

	if _, err := OpenRepository(ctx, name, namespace, repository, tempdir, GitRepositoryOptions{}); err == nil {
		t.Errorf("Unexpectedly succeeded opening empty repository with main branch validation enabled.")
	}

	if _, err := OpenRepository(ctx, name, namespace, repository, tempdir, GitRepositoryOptions{MainBranchStrategy: SkipVerification}); err != nil {
		t.Errorf("Failed to open empty git repository with main branch validation disabled: %v", err)
	}

	if _, err := OpenRepository(ctx, name, namespace, repository, tempdir, GitRepositoryOptions{MainBranchStrategy: CreateIfMissing}); err != nil {
		t.Errorf("Failed to create new main branch: %v", err)
	}
	_, err := repo.Reference(BranchName(g.branch).RefInRemote(), false)
	if err != nil {
		t.Errorf("Couldn't find branch %q after opening repository with CreateIfMissing strategy: %v", g.branch, err)
	}
}

// TestGitPackageRoundTrip creates a package in git and verifies we can read the contents back.
func (g GitSuite) TestGitPackageRoundTrip(t *testing.T) {
	tempdir := t.TempDir()
	p := filepath.Join(tempdir, "repo")
	serverRepo := InitEmptyRepositoryWithWorktree(t, p)

	if err := g.initRepo(serverRepo); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	ctx := context.Background()
	gitServerURL := ServeExistingRepository(t, serverRepo)

	// Now that we are running a git server, we can create a GitRepository backed by it

	const (
		repositoryName = "roundtrip"
		packageName    = "test-package"
		revision       = "v123"
		namespace      = "default"
	)
	spec := &configapi.GitRepository{
		Repo:   gitServerURL,
		Branch: g.branch,
	}

	root := filepath.Join(tempdir, "work")
	repo, err := OpenRepository(ctx, repositoryName, namespace, spec, root, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	// TODO: is there any state? should we  defer repo.Close()

	t.Logf("repo is %#v", repo)

	// Push a package to the repo

	wantResources := map[string]string{
		"hello": "world",
	}

	// We require a Kptfile to indicate the package boundary
	wantResources["Kptfile"] = "placeholder"

	{
		packageRevision := &v1alpha1.PackageRevision{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
			},
			Spec: v1alpha1.PackageRevisionSpec{
				PackageName:    packageName,
				Revision:       revision,
				RepositoryName: repositoryName,
			},
			Status: v1alpha1.PackageRevisionStatus{},
		}

		draft, err := repo.CreatePackageRevision(ctx, packageRevision)
		if err != nil {
			t.Fatalf("CreatePackageRevision(%#v) failed: %v", packageRevision, err)
		}

		newResources := &v1alpha1.PackageRevisionResources{}
		newResources.Spec.Resources = wantResources
		task := &v1alpha1.Task{}
		if err := draft.UpdateResources(ctx, newResources, task); err != nil {
			t.Fatalf("draft.UpdateResources(%#v, %#v) failed: %v", newResources, task, err)
		}

		revision, err := draft.Close(ctx)
		if err != nil {
			t.Fatalf("draft.Close() failed: %v", err)
		}
		klog.Infof("created revision %v", revision.KubeObjectName())
	}

	// We approve the draft so that we can fetch it
	{
		revisions, err := repo.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
		if err != nil {
			t.Fatalf("ListPackageRevisons failed: %v", err)
		}

		original := findPackage(t, revisions, repository.PackageRevisionKey{
			Repository: repositoryName,
			Package:    packageName,
			Revision:   revision,
		})

		update, err := repo.UpdatePackage(ctx, original)
		if err != nil {
			t.Fatalf("UpdatePackage(%#v failed: %v", original, err)
		}
		if err := update.UpdateLifecycle(ctx, v1alpha1.PackageRevisionLifecyclePublished); err != nil {
			t.Fatalf("UpdateLifecycle failed: %v", err)
		}
		approved, err := update.Close(ctx)
		if err != nil {
			t.Fatalf("Close() of %q, %q failed: %v", packageName, revision, err)
		}

		klog.Infof("approved revision %v", approved.KubeObjectName())
	}

	// Get the package again, the resources should match what we push
	{
		version := "v123"

		path := "test-package"
		packageRevision, gitLock, err := repo.GetPackage(ctx, version, path)
		if err != nil {
			t.Fatalf("GetPackage(%q, %q) failed: %v", version, path, err)
		}

		t.Logf("packageRevision is %s", packageRevision.KubeObjectName())
		t.Logf("gitLock is %#v", gitLock)

		resources, err := packageRevision.GetResources(ctx)
		if err != nil {
			t.Fatalf("GetResources() failed: %v", err)
		}

		t.Logf("resources is %v", resources.Spec.Resources)

		if !reflect.DeepEqual(resources.Spec.Resources, wantResources) {
			t.Fatalf("resources did not match expected; got %v, want %v", resources.Spec.Resources, wantResources)
		}
	}
}

// initRepo is a helper that creates a first commit, ensuring the repo is not empty.
func (g GitSuite) initRepo(repo *gogit.Repository) error {
	store := repo.Storer

	var objectHash plumbing.Hash
	{
		data := []byte("This is a test repo")
		eo := store.NewEncodedObject()
		eo.SetType(plumbing.BlobObject)
		eo.SetSize(int64(len(data)))

		w, err := eo.Writer()
		if err != nil {
			return fmt.Errorf("error creating object writer: %w", err)
		}

		if _, err = w.Write(data); err != nil {
			w.Close()
			return fmt.Errorf("error writing object data: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("error closing object data: %w", err)
		}

		if h, err := store.SetEncodedObject(eo); err != nil {
			return fmt.Errorf("error storing object: %w", err)
		} else {
			objectHash = h
		}
	}

	var treeHash plumbing.Hash
	{
		tree := object.Tree{}

		te := object.TreeEntry{
			Name: "README.md",
			Mode: filemode.Regular,
			Hash: objectHash,
		}
		tree.Entries = append(tree.Entries, te)

		eo := store.NewEncodedObject()
		if err := tree.Encode(eo); err != nil {
			return fmt.Errorf("error encoding tree: %w", err)
		}
		if h, err := store.SetEncodedObject(eo); err != nil {
			return fmt.Errorf("error storing tree: %w", err)
		} else {
			treeHash = h
		}
	}

	var commitHash plumbing.Hash
	{
		now := time.Now()
		commit := &object.Commit{
			Author: object.Signature{
				Name:  "Porch Author",
				Email: "author@kpt.dev",
				When:  now,
			},
			Committer: object.Signature{
				Name:  "Porch Committer",
				Email: "committer@kpt.dev",
				When:  now,
			},
			Message:  "First commit",
			TreeHash: treeHash,
		}

		eo := store.NewEncodedObject()
		if err := commit.Encode(eo); err != nil {
			return fmt.Errorf("error encoding commit: %w", err)
		}
		if h, err := store.SetEncodedObject(eo); err != nil {
			return fmt.Errorf("error storing commit: %w", err)
		} else {
			commitHash = h
		}
	}

	{
		name := plumbing.NewBranchReferenceName(g.branch)
		ref := plumbing.NewHashReference(name, commitHash)
		if err := repo.Storer.SetReference(ref); err != nil {
			return fmt.Errorf("error setting reference %q: %w", name, err)
		}

		// gogit uses suboptimal default reference name; delete it
		repo.Storer.RemoveReference(plumbing.Master)

		// create correct HEAD as a symbolic reference of the branch
		head := plumbing.NewSymbolicReference(plumbing.HEAD, name)
		if err := repo.Storer.SetReference(head); err != nil {
			return fmt.Errorf("error creating HEAD ref to %q: %w", ref, err)
		}
	}

	return nil
}

const Kptfile = `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: empty
info:
  description: Empty Package
`

func (g GitSuite) TestListPackagesTrivial(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "trivial-repository.tar")
	_, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

	ctx := context.Background()
	const (
		repositoryName = "empty"
		namespace      = "default"
	)

	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    g.branch,
		Directory: "/",
		SecretRef: configapi.SecretRef{},
	}, tempdir, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("Failed to list packages from %q: %v", tarfile, err)
	}
	if got, want := len(revisions), 0; got != want {
		t.Errorf("Number of packges in empty repository: got %d, want %d", got, want)
	}

	packageRevision := &v1alpha1.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: v1alpha1.PackageRevisionSpec{
			PackageName:    "test-package",
			Revision:       "v1",
			RepositoryName: repositoryName,
			Lifecycle:      v1alpha1.PackageRevisionLifecycleDraft,
		},
	}

	// Create a package draft
	draft, err := git.CreatePackageRevision(ctx, packageRevision)
	if err != nil {
		t.Fatalf("CreatePackageRevision() failed: %v", err)
	}
	resources := &v1alpha1.PackageRevisionResources{
		Spec: v1alpha1.PackageRevisionResourcesSpec{
			Resources: map[string]string{
				"Kptfile": Kptfile,
			},
		},
	}
	if err := draft.UpdateResources(ctx, resources, &v1alpha1.Task{
		Type: v1alpha1.TaskTypeInit,
		Init: &v1alpha1.PackageInitTaskSpec{
			Description: "Empty Package",
		},
	}); err != nil {
		t.Fatalf("UpdateResources() failed: %v", err)
	}
	newRevision, err := draft.Close(ctx)
	if err != nil {
		t.Fatalf("draft.Close() failed: %v", err)
	}

	result := newRevision.GetPackageRevision()
	if got, want := result.Spec.Lifecycle, v1alpha1.PackageRevisionLifecycleDraft; got != want {
		t.Errorf("Newly created package type: got %q, want %q", got, want)
	}

	// Verify
	verify, err := gogit.PlainOpen(filepath.Join(tempdir, ".git"))
	if err != nil {
		t.Fatalf("Failed to open git repository for verification: %v", err)
	}
	logRefs(t, verify, "Ref: ")
	draftRefName := plumbing.NewBranchReferenceName("drafts/test-package/v1")
	if _, err = verify.Reference(draftRefName, true); err != nil {
		t.Errorf("Failed to resolve %q references: %v", draftRefName, err)
	}
}

// trivial-repository.tar has a repon with a `main` branch and a single empty commit.
func (g GitSuite) TestCreatePackageInTrivialRepository(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "trivial-repository.tar")
	_, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

	ctx := context.Background()
	const (
		repositoryName = "trivial"
		namespace      = "default"
	)

	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    g.branch,
		Directory: "/",
		SecretRef: configapi.SecretRef{},
	}, tempdir, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("Failed to list packages from %q: %v", tarfile, err)
	}
	if got, want := len(revisions), 0; got != want {
		t.Errorf("Number of packges in the trivial repository: got %d, want %d", got, want)
	}

	packageRevision := &v1alpha1.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: v1alpha1.PackageRevisionSpec{
			PackageName:    "test-package",
			Revision:       "v1",
			RepositoryName: repositoryName,
			Lifecycle:      v1alpha1.PackageRevisionLifecycleDraft,
		},
	}

	// Create a package draft
	draft, err := git.CreatePackageRevision(ctx, packageRevision)
	if err != nil {
		t.Fatalf("CreatePackageRevision() failed: %v", err)
	}
	resources := &v1alpha1.PackageRevisionResources{
		Spec: v1alpha1.PackageRevisionResourcesSpec{
			Resources: map[string]string{
				"Kptfile": Kptfile,
			},
		},
	}
	if err := draft.UpdateResources(ctx, resources, &v1alpha1.Task{
		Type: v1alpha1.TaskTypeInit,
		Init: &v1alpha1.PackageInitTaskSpec{
			Description: "Empty Package",
		},
	}); err != nil {
		t.Fatalf("UpdateResources() failed: %v", err)
	}
	newRevision, err := draft.Close(ctx)
	if err != nil {
		t.Fatalf("draft.Close() failed: %v", err)
	}

	result := newRevision.GetPackageRevision()
	if got, want := result.Spec.Lifecycle, v1alpha1.PackageRevisionLifecycleDraft; got != want {
		t.Errorf("Newly created package type: got %q, want %q", got, want)
	}
}

func (g GitSuite) TestListPackagesSimple(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "simple-repository.tar")
	_, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

	ctx := context.Background()
	const (
		repositoryName = "simple"
		namespace      = "default"
	)

	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    g.branch,
		Directory: "/",
		SecretRef: configapi.SecretRef{},
	}, tempdir, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("Failed to list packages from %q: %v", tarfile, err)
	}

	want := map[repository.PackageRevisionKey]v1alpha1.PackageRevisionLifecycle{
		{Repository: "simple", Package: "empty", Revision: "v1"}:   v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "simple", Package: "basens", Revision: "v1"}:  v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "simple", Package: "basens", Revision: "v2"}:  v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "simple", Package: "istions", Revision: "v1"}: v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "simple", Package: "istions", Revision: "v2"}: v1alpha1.PackageRevisionLifecyclePublished,

		// TODO: may want to filter these out, for example by including only those package
		// revisions from main branch that differ in content (their tree hash) from another
		// taged revision of the package.
		{Repository: "simple", Package: "empty", Revision: g.branch}:   v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "simple", Package: "basens", Revision: g.branch}:  v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "simple", Package: "istions", Revision: g.branch}: v1alpha1.PackageRevisionLifecyclePublished,
	}

	got := map[repository.PackageRevisionKey]v1alpha1.PackageRevisionLifecycle{}
	for _, r := range revisions {
		rev := r.GetPackageRevision()
		got[repository.PackageRevisionKey{
			Repository: rev.Spec.RepositoryName,
			Package:    rev.Spec.PackageName,
			Revision:   rev.Spec.Revision,
		}] = rev.Spec.Lifecycle
	}

	if !cmp.Equal(want, got) {
		t.Errorf("Package Revisions in simple-repository: (-want,+got): %s", cmp.Diff(want, got))
	}
}

func (g GitSuite) TestListPackagesDrafts(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "drafts-repository.tar")
	_, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

	ctx := context.Background()
	const (
		repositoryName = "drafts"
		namespace      = "default"
	)

	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    g.branch,
		Directory: "/",
		SecretRef: configapi.SecretRef{},
	}, tempdir, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("Failed to list packages from %q: %v", tarfile, err)
	}

	want := map[repository.PackageRevisionKey]v1alpha1.PackageRevisionLifecycle{
		{Repository: "drafts", Package: "empty", Revision: "v1"}:   v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "drafts", Package: "basens", Revision: "v1"}:  v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "drafts", Package: "basens", Revision: "v2"}:  v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "drafts", Package: "istions", Revision: "v1"}: v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "drafts", Package: "istions", Revision: "v2"}: v1alpha1.PackageRevisionLifecyclePublished,

		{Repository: "drafts", Package: "bucket", Revision: "v1"}:           v1alpha1.PackageRevisionLifecycleDraft,
		{Repository: "drafts", Package: "none", Revision: "v1"}:             v1alpha1.PackageRevisionLifecycleDraft,
		{Repository: "drafts", Package: "pkg-with-history", Revision: "v1"}: v1alpha1.PackageRevisionLifecycleDraft,

		// TODO: filter main branch out? see above
		{Repository: "drafts", Package: "basens", Revision: g.branch}:  v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "drafts", Package: "empty", Revision: g.branch}:   v1alpha1.PackageRevisionLifecyclePublished,
		{Repository: "drafts", Package: "istions", Revision: g.branch}: v1alpha1.PackageRevisionLifecyclePublished,
	}

	got := map[repository.PackageRevisionKey]v1alpha1.PackageRevisionLifecycle{}
	for _, r := range revisions {
		rev := r.GetPackageRevision()
		got[repository.PackageRevisionKey{
			Repository: rev.Spec.RepositoryName,
			Package:    rev.Spec.PackageName,
			Revision:   rev.Spec.Revision,
		}] = rev.Spec.Lifecycle
	}

	if !cmp.Equal(want, got) {
		t.Errorf("Package Revisions in drafts-repository: (-want,+got): %s", cmp.Diff(want, got))
	}
}

func (g GitSuite) TestApproveDraft(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "drafts-repository.tar")
	repo, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

	const (
		repositoryName                            = "approve"
		namespace                                 = "default"
		draft              BranchName             = "drafts/bucket/v1"
		finalReferenceName plumbing.ReferenceName = "refs/tags/bucket/v1"
	)
	ctx := context.Background()
	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    g.branch,
		Directory: "/",
	}, tempdir, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("ListPackageRevisions failed: %v", err)
	}

	bucket := findPackage(t, revisions, repository.PackageRevisionKey{
		Repository: repositoryName,
		Package:    "bucket",
		Revision:   "v1",
	})

	// Before Update; Check server references. Draft must exist, final not.
	refMustExist(t, repo, draft.RefInRemote())
	refMustNotExist(t, repo, finalReferenceName)

	update, err := git.UpdatePackage(ctx, bucket)
	if err != nil {
		t.Fatalf("UpdatePackage failed: %v", err)
	}

	update.UpdateLifecycle(ctx, v1alpha1.PackageRevisionLifecyclePublished)

	new, err := update.Close(ctx)
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	rev := new.GetPackageRevision()
	if got, want := rev.Spec.Lifecycle, v1alpha1.PackageRevisionLifecyclePublished; got != want {
		t.Errorf("Approved package lifecycle: got %s, want %s", got, want)
	}

	// After Update: Final must exist, draft must not exist
	refMustNotExist(t, repo, draft.RefInRemote())
	refMustExist(t, repo, finalReferenceName)
}

func (g GitSuite) TestApproveDraft2(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "drafts-repository.tar")
	repo, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

	const (
		repositoryName                            = "approve"
		namespace                                 = "default"
		draft              BranchName             = "drafts/pkg-with-history/v1"
		finalReferenceName plumbing.ReferenceName = "refs/tags/pkg-with-history/v1"
	)
	ctx := context.Background()
	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    g.branch,
		Directory: "/",
	}, tempdir, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("ListPackageRevisions failed: %v", err)
	}

	bucket := findPackage(t, revisions, repository.PackageRevisionKey{
		Repository: repositoryName,
		Package:    "pkg-with-history",
		Revision:   "v1",
	})

	// Before Update; Check server references. Draft must exist, final not.
	refMustExist(t, repo, draft.RefInRemote())
	refMustNotExist(t, repo, finalReferenceName)

	update, err := git.UpdatePackage(ctx, bucket)
	if err != nil {
		t.Fatalf("UpdatePackage failed: %v", err)
	}

	update.UpdateLifecycle(ctx, v1alpha1.PackageRevisionLifecyclePublished)

	new, err := update.Close(ctx)
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	rev := new.GetPackageRevision()
	if got, want := rev.Spec.Lifecycle, v1alpha1.PackageRevisionLifecyclePublished; got != want {
		t.Errorf("Approved package lifecycle: got %s, want %s", got, want)
	}
	if got, want := len(rev.Spec.Tasks), 4; got != want {
		t.Errorf("Approved package task count: got %d, want %d", got, want)
	}

	// After Update: Final must exist, draft must not exist
	refMustNotExist(t, repo, draft.RefInRemote())
	refMustExist(t, repo, finalReferenceName)
}

func (g GitSuite) TestDeletePackages(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "drafts-repository.tar")
	repo, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

	const (
		repositoryName = "delete"
		namespace      = "delete-namespace"
	)

	ctx := context.Background()
	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:   address,
		Branch: g.branch,
	}, tempdir, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("OpenRepository(%q) failed: %v", address, err)
	}

	// If we delete one of these packages, we expect the reference to be deleted too
	wantDeletedRefs := map[repository.PackageRevisionKey]plumbing.ReferenceName{
		{Repository: "delete", Package: "bucket", Revision: "v1"}:  "refs/heads/drafts/bucket/v1",
		{Repository: "delete", Package: "none", Revision: "v1"}:    "refs/heads/drafts/none/v1",
		{Repository: "delete", Package: "basens", Revision: "v1"}:  "refs/tags/basens/v1",
		{Repository: "delete", Package: "basens", Revision: "v2"}:  "refs/tags/basens/v2",
		{Repository: "delete", Package: "empty", Revision: "v1"}:   "refs/tags/empty/v1",
		{Repository: "delete", Package: "istions", Revision: "v1"}: "refs/tags/istions/v1",
		{Repository: "delete", Package: "istions", Revision: "v2"}: "refs/tags/istions/v2",
	}

	// Delete all packages
	all, err := git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("ListPackageRevisions failed: %v", err)
	}

	for len(all) > 0 {
		// Delete one of the packages
		deleting := all[0]
		pr := deleting.GetPackageRevision()
		name := repository.PackageRevisionKey{Repository: pr.Spec.RepositoryName, Package: pr.Spec.PackageName, Revision: pr.Spec.Revision}

		if rn, ok := wantDeletedRefs[name]; ok {
			// Verify the reference still exists
			refMustExist(t, repo, rn)
		}

		if err := git.DeletePackageRevision(ctx, deleting); err != nil {
			t.Fatalf("DeletePackageRevision(%q) failed: %v", deleting.KubeObjectName(), err)
		}

		if rn, ok := wantDeletedRefs[name]; ok {
			// Verify the reference no longer exists
			refMustNotExist(t, repo, rn)
		}

		// Re-list packages and check the deleted package is absent
		all, err = git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
		if err != nil {
			t.Fatalf("ListPackageRevisions failed: %v", err)
		}

		packageMustNotExist(t, all, name)
	}

	// The only got references should be main and HEAD
	got := map[plumbing.ReferenceName]bool{}
	forEachRef(t, repo, func(ref *plumbing.Reference) error {
		got[ref.Name()] = true
		return nil
	})

	// branch may be `refs/heads/main` for some test runs
	branch := plumbing.NewBranchReferenceName(g.branch)
	want := map[plumbing.ReferenceName]bool{
		branch:                   true,
		DefaultMainReferenceName: true,
		"HEAD":                   true,
	}
	if !cmp.Equal(want, got) {
		t.Fatalf("Unexpected references after deleting all packages (-want, +got): %s", cmp.Diff(want, got))
	}

	// And there should be no packages in main branch
	main := resolveReference(t, repo, branch)
	tree := getCommitTree(t, repo, main.Hash())
	if len(tree.Entries) > 0 {
		var b bytes.Buffer
		for i := range tree.Entries {
			e := &tree.Entries[i]
			fmt.Fprintf(&b, "%s: %s (%s)", e.Name, e.Hash, e.Mode)
		}
		// Tree is not empty after deleting all packages
		t.Errorf("%q branch has non-empty tree after all packages have been deleted: %s", branch, b.String())
	}
}

// Test introduces package in the upstream repo and lists is after refresh.
func (g GitSuite) TestRefreshRepo(t *testing.T) {
	upstreamDir := t.TempDir()
	downstreamDir := t.TempDir()
	tarfile := filepath.Join("testdata", "simple-repository.tar")
	upstream := OpenGitRepositoryFromArchiveWithWorktree(t, tarfile, upstreamDir)
	InitializeBranch(t, upstream, g.branch)
	address := ServeExistingRepository(t, upstream)

	const (
		repositoryName = "refresh"
		namespace      = "refresh-namespace"
	)

	newPackageName := repository.PackageRevisionKey{
		Repository: "refresh",
		Package:    "newpkg",
		Revision:   "v3",
	}

	ctx := context.Background()
	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo: address,
	}, downstreamDir, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("OpenRepository(%q) failed: %v", address, err)
	}

	all, err := git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("ListPackageRevisions failed: %v", err)
	}

	// Confirm we listed some package(s)
	findPackage(t, all, repository.PackageRevisionKey{Repository: "refresh", Package: "basens", Revision: "v2"})
	packageMustNotExist(t, all, newPackageName)

	// Create package in the upstream repository
	wt, err := upstream.Worktree()
	if err != nil {
		t.Fatalf("Worktree failed: %v", err)
	}

	name := plumbing.NewBranchReferenceName(g.branch)
	main := resolveReference(t, upstream, name)
	if err := wt.Checkout(&gogit.CheckoutOptions{
		Branch: main.Name(),
		Force:  true,
	}); err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	const kptfileName = "newpkg/Kptfile"
	file, err := wt.Filesystem.Create(kptfileName)
	if err != nil {
		t.Fatalf("Filesystem.Create failed: %v", err)
	}
	if _, err := file.Write([]byte(Kptfile)); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if _, err := wt.Add(kptfileName); err != nil {
		t.Fatalf("Failed to add file to index: %v", err)
	}
	sig := object.Signature{
		Name:  "Test",
		Email: "test@kpt.dev",
		When:  time.Now(),
	}
	commit, err := wt.Commit("Hello", &gogit.CommitOptions{
		Author:    &sig,
		Committer: &sig,
	})
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	tag := plumbing.NewHashReference(plumbing.NewTagReferenceName("newpkg/v3"), commit)
	if err := upstream.Storer.SetReference(tag); err != nil {
		t.Fatalf("Failed to create tag %s: %v", tag, err)
	}

	all, err = git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("ListPackageRevisions(Refresh) failed; %v", err)
	}
	findPackage(t, all, newPackageName)
}

// The test deletes packages on the upstream one by one and validates they were
// pruned in the registered repository on refresh.
func (g GitSuite) TestPruneRemotes(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "drafts-repository.tar")
	repo, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

	const (
		name      = "prune"
		namespace = "prune-namespace"
	)

	ctx := context.Background()
	git, err := OpenRepository(ctx, name, namespace, &configapi.GitRepository{
		Repo:   address,
		Branch: g.branch,
	}, tempdir, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("OpenRepository(%q) failed: %v", address, err)
	}

	for _, pair := range []struct {
		ref plumbing.ReferenceName
		pkg repository.PackageRevisionKey
	}{
		{
			ref: "refs/heads/drafts/bucket/v1",
			pkg: repository.PackageRevisionKey{Repository: "prune", Package: "bucket", Revision: "v1"},
		},
		{
			ref: "refs/heads/drafts/none/v1",
			pkg: repository.PackageRevisionKey{Repository: "prune", Package: "none", Revision: "v1"},
		},
		{
			ref: "refs/tags/basens/v1",
			pkg: repository.PackageRevisionKey{Repository: "prune", Package: "basens", Revision: "v1"},
		},
		{
			ref: "refs/tags/basens/v2",
			pkg: repository.PackageRevisionKey{Repository: "prune", Package: "basens", Revision: "v2"},
		},
		{
			ref: "refs/tags/empty/v1",
			pkg: repository.PackageRevisionKey{Repository: "prune", Package: "empty", Revision: "v1"},
		},
		{
			ref: "refs/tags/istions/v1",
			pkg: repository.PackageRevisionKey{Repository: "prune", Package: "istions", Revision: "v1"},
		},
		{
			ref: "refs/tags/istions/v2",
			pkg: repository.PackageRevisionKey{Repository: "prune", Package: "istions", Revision: "v2"},
		},
	} {
		repositoryMustHavePackageRevision(t, git, pair.pkg)
		refMustExist(t, repo, pair.ref)
		if err := repo.Storer.RemoveReference(pair.ref); err != nil {
			t.Fatalf("RemoveReference(%q) failed: %v", pair.ref, err)
		}
		repositoryMustNotHavePackageRevision(t, git, pair.pkg)
	}
}

func (g GitSuite) TestNested(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "nested-repository.tar")
	repo, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

	ctx := context.Background()
	const (
		repositoryName = "nested"
		namespace      = "default"
	)

	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    g.branch,
		Directory: "/",
	}, tempdir, GitRepositoryOptions{})
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("Failed to list packages from %q: %v", tarfile, err)
	}

	// Name of the registered branch
	branch := plumbing.NewBranchReferenceName(g.branch)

	// Check that all tags and branches have their packages.
	want := map[string]v1alpha1.PackageRevisionLifecycle{}
	forEachRef(t, repo, func(ref *plumbing.Reference) error {
		switch name := string(ref.Name()); {
		case strings.HasPrefix(name, tagsPrefixInRemoteRepo):
			want[strings.TrimPrefix(name, tagsPrefixInRemoteRepo)] = v1alpha1.PackageRevisionLifecyclePublished
		case strings.HasPrefix(name, draftsPrefixInRemoteRepo):
			want[strings.TrimPrefix(name, draftsPrefixInRemoteRepo)] = v1alpha1.PackageRevisionLifecycleDraft
		case strings.HasPrefix(name, proposedPrefixInRemoteRepo):
			want[strings.TrimPrefix(name, proposedPrefixInRemoteRepo)] = v1alpha1.PackageRevisionLifecycleProposed
		case name == string(branch):
			// Skip the registered 'main' branch.
		case name == string(DefaultMainReferenceName), name == "HEAD":
			// skip main and HEAD
		default:
			// There should be no other refs in the repository.
			return fmt.Errorf("unexpected reference: %s", ref)
		}
		return nil
	})

	got := map[string]v1alpha1.PackageRevisionLifecycle{}
	for _, pr := range revisions {
		rev := pr.GetPackageRevision()
		if rev.Spec.Revision == g.branch {
			// skip packages with the revision of the main registered branch,
			// to match the above simplified package discovery algo.
			continue
		}
		got[fmt.Sprintf("%s/%s", rev.Spec.PackageName, rev.Spec.Revision)] = rev.Spec.Lifecycle
	}

	if !cmp.Equal(want, got) {
		t.Errorf("Discovered packages differ: (-want,+got): %s", cmp.Diff(want, got))
	}
}

func createPackageRevisionMap(revisions []repository.PackageRevision) map[string]bool {
	result := map[string]bool{}
	for _, pr := range revisions {
		key := pr.Key()
		result[fmt.Sprintf("%s/%s", key.Package, key.Revision)] = true
	}
	return result
}

func sliceToSet(s []string) map[string]bool {
	result := map[string]bool{}
	for _, v := range s {
		result[v] = true
	}
	return result
}

func (g GitSuite) TestNestedDirectories(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		directory string
		packages  []string
	}{
		{
			directory: "sample",
			packages:  []string{"sample/v1", "sample/v2", "sample/" + g.branch},
		},
		{
			directory: "nonexistent",
			packages:  []string{},
		},
		{
			directory: "catalog/gcp",
			packages: []string{
				"catalog/gcp/cloud-sql/v1",
				"catalog/gcp/spanner/v1",
				"catalog/gcp/bucket/v2",
				"catalog/gcp/bucket/v1",
				"catalog/gcp/bucket/" + g.branch,
			},
		},
	} {
		t.Run(tc.directory, func(t *testing.T) {
			tempdir := t.TempDir()
			tarfile := filepath.Join("testdata", "nested-repository.tar")
			_, address := ServeGitRepositoryWithBranch(t, tarfile, tempdir, g.branch)

			const (
				repositoryName = "directory"
				namespace      = "default"
			)

			git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
				Repo:      address,
				Branch:    g.branch,
				Directory: tc.directory,
			}, tempdir, GitRepositoryOptions{})
			if err != nil {
				t.Fatalf("Failed to open Git repository loaded from %q with directory %q: %v", tarfile, tc.directory, err)
			}

			revisions, err := git.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
			if err != nil {
				t.Fatalf("Failed to list packages from %q: %v", tarfile, err)
			}

			got := createPackageRevisionMap(revisions)
			want := sliceToSet(tc.packages)

			if !cmp.Equal(want, got) {
				t.Errorf("Packages rooted in %q; Unexpected result (-want,+got): %s", tc.directory, cmp.Diff(want, got))
			}
		})
	}
}
