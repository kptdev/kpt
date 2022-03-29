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
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
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

// TestGitPackageRoundTrip creates a package in git and verifies we can read the contents back.
func TestGitPackageRoundTrip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempdir := t.TempDir()

	// Start a mock git server
	gitServerAddressChannel := make(chan net.Addr)

	p := filepath.Join(tempdir, "repo")
	serverRepo, err := gogit.PlainInit(p, true)
	if err != nil {
		t.Fatalf("failed to open source repo %q: %v", p, err)
	}

	if err := initRepo(serverRepo); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	gitServer, err := NewGitServer(serverRepo)
	if err != nil {
		t.Fatalf("NewGitServer() failed: %v", err)
	}

	go func() {
		if err := gitServer.ListenAndServe(ctx, "127.0.0.1:0", gitServerAddressChannel); err != nil {
			if ctx.Err() == nil {
				t.Errorf("ListenAndServe failed: %v", err)
			}
		}
	}()

	gitServerAddress, ok := <-gitServerAddressChannel
	if !ok {
		t.Fatalf("could not get address from server")
	}

	// Now that we are running a git server, we can create a GitRepository backed by it

	gitServerURL := "http://" + gitServerAddress.String()
	name := ""
	namespace := ""
	spec := &configapi.GitRepository{
		Repo: gitServerURL,
	}

	var credentialResolver repository.CredentialResolver
	root := filepath.Join(tempdir, "work")

	repo, err := OpenRepository(ctx, name, namespace, spec, credentialResolver, root)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	// TODO: is there any state? should we  defer repo.Close()

	t.Logf("repo is %#v", repo)

	// Push a package to the repo
	packageName := "test-package"
	revision := "v123"

	wantResources := map[string]string{
		"hello": "world",
	}

	{
		packageRevision := &v1alpha1.PackageRevision{}
		packageRevision.Spec.PackageName = packageName
		packageRevision.Spec.Revision = revision

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
		klog.Infof("created revision %v", revision.Name())
	}

	// We approve the draft so that we can fetch it
	{
		approved, err := repo.(*gitRepository).ApprovePackageRevision(ctx, packageName, revision)
		if err != nil {
			t.Fatalf("ApprovePackageRevision(%q, %q) failed: %v", packageName, revision, err)
		}

		klog.Infof("approved revision %v", approved.Name())
	}

	// We reopen to refetch
	// TODO: This is pretty hacky...
	repo, err = OpenRepository(ctx, name, namespace, spec, credentialResolver, root)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	// TODO: is there any state? should we  defer repo.Close()

	// Get the package again, the resources should match what we push
	{
		version := "v123"

		path := "test-package"
		packageRevision, gitLock, err := repo.GetPackage(version, path)
		if err != nil {
			t.Fatalf("GetPackage(%q, %q) failed: %v", version, path, err)
		}

		t.Logf("packageRevision is %s", packageRevision.Name())
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
func initRepo(repo *gogit.Repository) error {
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
		ref := plumbing.NewHashReference(Main, commitHash)
		if err := repo.Storer.SetReference(ref); err != nil {
			return fmt.Errorf("error setting reference %q: %w", Main, err)
		}

		// gogit uses suboptimal default reference name; delete it
		repo.Storer.RemoveReference(plumbing.Master)

		// create correct HEAD as a symbolic reference of main branch
		head := plumbing.NewSymbolicReference(plumbing.HEAD, Main)
		if err := repo.Storer.SetReference(head); err != nil {
			return fmt.Errorf("error creating HEAD ref: %w", err)
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

func TestListPackagesEmpty(t *testing.T) {
	testdata := TestDataAbs(t, "testdata")
	tempdir := t.TempDir()
	tarfile := filepath.Join(testdata, "empty-repository.tar")
	_, address := ServeGitRepository(t, tarfile, tempdir)

	ctx := context.Background()
	const (
		repositoryName = "empty"
		namespace      = "default"
	)
	var resolver repository.CredentialResolver

	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    "main",
		Directory: "/",
		SecretRef: configapi.SecretRef{},
	}, resolver, tempdir)
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx)
	if err != nil {
		t.Fatalf("Failed to list packages from %q: %v", tarfile, err)
	}
	if got, want := len(revisions), 0; got != want {
		t.Errorf("Number of packges in empty repository: got %d, want %d", got, want)
	}

	packageRevision := &v1alpha1.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "empty:test-packgae:v1",
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

	result, err := newRevision.GetPackageRevision()
	if err != nil {
		t.Fatalf("GetPackageRevision() failed: %v", err)
	}
	if got, want := result.Spec.Lifecycle, v1alpha1.PackageRevisionLifecycleDraft; got != want {
		t.Errorf("Newly created package type: got %q, want %q", got, want)
	}

	// Verify
	verify, err := gogit.PlainOpen(filepath.Join(tempdir, ".git"))
	if err != nil {
		t.Fatalf("Failed to open git repository for verification: %v", err)
	}
	forEachRef(t, verify, func(ref *plumbing.Reference) error {
		t.Logf("Ref: %s", ref)
		return nil
	})
	draftRefName := plumbing.NewBranchReferenceName("drafts/test-package/v1")
	if _, err = verify.Reference(draftRefName, true); err != nil {
		t.Errorf("Failed to resolve %q references: %v", draftRefName, err)
	}
}

// trivial-repository.tar has a repon with a `main` branch and a single empty commit.
func TestCreatePackageInTrivialRepository(t *testing.T) {
	testdata := TestDataAbs(t, "testdata")
	tempdir := t.TempDir()
	tarfile := filepath.Join(testdata, "trivial-repository.tar")
	_, address := ServeGitRepository(t, tarfile, tempdir)

	ctx := context.Background()
	const (
		repositoryName = "trivial"
		namespace      = "default"
	)
	var resolver repository.CredentialResolver

	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    "main",
		Directory: "/",
		SecretRef: configapi.SecretRef{},
	}, resolver, tempdir)
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx)
	if err != nil {
		t.Fatalf("Failed to list packages from %q: %v", tarfile, err)
	}
	if got, want := len(revisions), 0; got != want {
		t.Errorf("Number of packges in the trivial repository: got %d, want %d", got, want)
	}

	packageRevision := &v1alpha1.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "trivial:test-packgae:v1",
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

	result, err := newRevision.GetPackageRevision()
	if err != nil {
		t.Fatalf("GetPackageRevision() failed: %v", err)
	}
	if got, want := result.Spec.Lifecycle, v1alpha1.PackageRevisionLifecycleDraft; got != want {
		t.Errorf("Newly created package type: got %q, want %q", got, want)
	}
}

func TestListPackagesSimple(t *testing.T) {
	testdata := TestDataAbs(t, "testdata")
	tempdir := t.TempDir()
	tarfile := filepath.Join(testdata, "simple-repository.tar")
	_, address := ServeGitRepository(t, tarfile, tempdir)

	ctx := context.Background()
	const (
		repositoryName = "simple"
		namespace      = "default"
	)
	var resolver repository.CredentialResolver

	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    "main",
		Directory: "/",
		SecretRef: configapi.SecretRef{},
	}, resolver, tempdir)
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx)
	if err != nil {
		t.Fatalf("Failed to list packages from %q: %v", tarfile, err)
	}

	want := map[string]v1alpha1.PackageRevisionLifecycle{
		"simple:empty:v1":   v1alpha1.PackageRevisionLifecycleFinal,
		"simple:basens:v1":  v1alpha1.PackageRevisionLifecycleFinal,
		"simple:basens:v2":  v1alpha1.PackageRevisionLifecycleFinal,
		"simple:istions:v1": v1alpha1.PackageRevisionLifecycleFinal,
		"simple:istions:v2": v1alpha1.PackageRevisionLifecycleFinal,

		// TODO: may want to filter these out, for example by including only those package
		// revisions from main branch that differ in content (their tree hash) from another
		// taged revision of the package.
		"simple:empty:main":   v1alpha1.PackageRevisionLifecycleFinal,
		"simple:basens:main":  v1alpha1.PackageRevisionLifecycleFinal,
		"simple:istions:main": v1alpha1.PackageRevisionLifecycleFinal,
	}

	got := map[string]v1alpha1.PackageRevisionLifecycle{}
	for _, r := range revisions {
		rev, err := r.GetPackageRevision()
		if err != nil {
			t.Errorf("GetPackageRevision failed for %q: %v", r.Name(), err)
		}
		got[r.Name()] = rev.Spec.Lifecycle
	}

	if !cmp.Equal(want, got) {
		t.Errorf("Package Revisions in simple-repository: (-want,+got): %s", cmp.Diff(want, got))
	}
}

func TestListPackagesDrafts(t *testing.T) {
	testdata := TestDataAbs(t, "testdata")
	tempdir := t.TempDir()
	tarfile := filepath.Join(testdata, "drafts-repository.tar")
	_, address := ServeGitRepository(t, tarfile, tempdir)

	ctx := context.Background()
	const (
		repositoryName = "drafts"
		namespace      = "default"
	)
	var resolver repository.CredentialResolver

	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    "main",
		Directory: "/",
		SecretRef: configapi.SecretRef{},
	}, resolver, tempdir)
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx)
	if err != nil {
		t.Fatalf("Failed to list packages from %q: %v", tarfile, err)
	}

	want := map[string]v1alpha1.PackageRevisionLifecycle{
		"drafts:empty:v1":   v1alpha1.PackageRevisionLifecycleFinal,
		"drafts:basens:v1":  v1alpha1.PackageRevisionLifecycleFinal,
		"drafts:basens:v2":  v1alpha1.PackageRevisionLifecycleFinal,
		"drafts:istions:v1": v1alpha1.PackageRevisionLifecycleFinal,
		"drafts:istions:v2": v1alpha1.PackageRevisionLifecycleFinal,

		"drafts:bucket:v1": v1alpha1.PackageRevisionLifecycleDraft,
		"drafts:none:v1":   v1alpha1.PackageRevisionLifecycleDraft,

		// TODO: filter main branch out? see above
		"drafts:basens:main":  v1alpha1.PackageRevisionLifecycleFinal,
		"drafts:empty:main":   v1alpha1.PackageRevisionLifecycleFinal,
		"drafts:istions:main": v1alpha1.PackageRevisionLifecycleFinal,
	}

	got := map[string]v1alpha1.PackageRevisionLifecycle{}
	for _, r := range revisions {
		rev, err := r.GetPackageRevision()
		if err != nil {
			t.Errorf("GetPackageRevision failed for %q: %v", r.Name(), err)
		}
		got[r.Name()] = rev.Spec.Lifecycle
	}

	if !cmp.Equal(want, got) {
		t.Errorf("Package Revisions in drafts-repository: (-want,+got): %s", cmp.Diff(want, got))
	}
}

func TestApproveDraft(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "drafts-repository.tar")
	repo, address := ServeGitRepository(t, tarfile, tempdir)

	const (
		repositoryName                            = "approve"
		namespace                                 = "default"
		draftReferenceName plumbing.ReferenceName = "refs/heads/drafts/bucket/v1"
		finalReferenceName plumbing.ReferenceName = "refs/tags/bucket/v1"
	)
	ctx := context.Background()
	var resolver repository.CredentialResolver
	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo:      address,
		Branch:    "main",
		Directory: "/",
	}, resolver, tempdir)
	if err != nil {
		t.Fatalf("Failed to open Git repository loaded from %q: %v", tarfile, err)
	}

	revisions, err := git.ListPackageRevisions(ctx)
	if err != nil {
		t.Fatalf("ListPackageRevisions failed: %v", err)
	}

	bucket := findPackage(t, revisions, "approve:bucket:v1")

	// Before Update; Check server references. Draft must exist, final not.
	refMustExist(t, repo, draftReferenceName)
	refMustNotExist(t, repo, finalReferenceName)

	update, err := git.UpdatePackage(ctx, bucket)
	if err != nil {
		t.Fatalf("UpdatePackage failed: %v", err)
	}

	update.UpdateLifecycle(ctx, v1alpha1.PackageRevisionLifecycleFinal)

	new, err := update.Close(ctx)
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	rev, err := new.GetPackageRevision()
	if err != nil {
		t.Fatalf("GetPackageRevision failed: %v", err)
	}

	if got, want := rev.Spec.Lifecycle, v1alpha1.PackageRevisionLifecycleFinal; got != want {
		t.Errorf("Approved package lifecycle: got %s, want %s", got, want)
	}

	// After Update: Final must exist, draft must not exist
	refMustNotExist(t, repo, draftReferenceName)
	refMustExist(t, repo, finalReferenceName)
}

func TestDeletePackages(t *testing.T) {
	tempdir := t.TempDir()
	tarfile := filepath.Join("testdata", "drafts-repository.tar")
	repo, address := ServeGitRepository(t, tarfile, tempdir)

	const (
		repositoryName = "delete"
		namespace      = "delete-namespace"
	)

	ctx := context.Background()
	var resolver repository.CredentialResolver
	git, err := OpenRepository(ctx, repositoryName, namespace, &configapi.GitRepository{
		Repo: address,
	}, resolver, tempdir)
	if err != nil {
		t.Fatalf("OpenRepository(%q) failed: %v", address, err)
	}

	type PackageReference struct {
		name string
		ref  plumbing.ReferenceName
	}

	// If we delete one of these packages, we expect the reference to be deleted too
	wantDeletedRefs := map[string]plumbing.ReferenceName{
		"delete:bucket:v1":  "refs/heads/drafts/bucket/v1",
		"delete:none:v1":    "refs/heads/drafts/none/v1",
		"delete:basens:v1":  "refs/tags/basens/v1",
		"delete:basens:v2":  "refs/tags/basens/v2",
		"delete:empty:v1":   "refs/tags/empty/v1",
		"delete:istions:v1": "refs/tags/istions/v1",
		"delete:istions:v2": "refs/tags/istions/v2",
	}

	// Delete all packages
	all, err := git.ListPackageRevisions(ctx)
	if err != nil {
		t.Fatalf("ListPackageRevisions failed: %v", err)
	}

	for len(all) > 0 {
		// Delete one of the packages
		deleting := all[0]
		name := deleting.Name()

		if rn, ok := wantDeletedRefs[name]; ok {
			// Verify the reference still exists
			refMustExist(t, repo, rn)
		}

		if err := git.DeletePackageRevision(ctx, deleting); err != nil {
			t.Fatalf("DeletePackageRevision(%q) failed: %v", deleting.Name(), err)
		}

		if rn, ok := wantDeletedRefs[name]; ok {
			// Verify the reference no longer exists
			refMustNotExist(t, repo, rn)
		}

		// Re-list packages and check the deleted package is absent
		all, err = git.ListPackageRevisions(ctx)
		if err != nil {
			t.Fatalf("ListPackageRevisions failed: %v", err)
		}

		for _, existing := range all {
			if existing.Name() == name {
				t.Errorf("Deleted package %q was found among the list results", name)
			}
		}
	}

	// The only got references should be main and HEAD
	got := map[plumbing.ReferenceName]bool{}
	forEachRef(t, repo, func(ref *plumbing.Reference) error {
		got[ref.Name()] = true
		return nil
	})
	want := map[plumbing.ReferenceName]bool{
		refMain: true,
		"HEAD":  true,
	}
	if !cmp.Equal(want, got) {
		t.Fatalf("Unexpected references after deleting all packages (-want, +got): %s", cmp.Diff(want, got))
	}

	// And there should be no packages in main branch
	main := resolveReference(t, repo, refMain)
	tree := getCommitTree(t, repo, main.Hash())
	if len(tree.Entries) > 0 {
		var b bytes.Buffer
		for i := range tree.Entries {
			e := &tree.Entries[i]
			fmt.Fprintf(&b, "%s: %s (%s)", e.Name, e.Hash, e.Mode)
		}
		// Tree is not empty after deleting all packages
		t.Errorf("%q branch has non-empty tree after all packages have been deleted: %s", refMain, b.String())
	}
}
