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
	"context"
	"flag"
	"fmt"
	"io/ioutil"
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

	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("TempDir failed: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempdir); err != nil {
			t.Errorf("RemoveAll(%q) failed: %v", tempdir, err)
		}
	}()

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
