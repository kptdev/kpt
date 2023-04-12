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

package engine

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/git"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

func createRepoWithContents(t *testing.T, contentDir string) *gogit.Repository {
	repo, err := gogit.Init(memory.NewStorage(), memfs.New())
	if err != nil {
		t.Fatalf("Failed to initialize in-memory git repository: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get git repository worktree: %v", err)
	}

	if err := filepath.Walk(contentDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		} else if !info.Mode().IsRegular() {
			return fmt.Errorf("irregular file object detected: %q (%s)", path, info.Mode())
		}
		rel, err := filepath.Rel(contentDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path from %q to %q: %w", contentDir, path, err)
		}
		dir := filepath.Dir(rel)
		if err := wt.Filesystem.MkdirAll(dir, 0777); err != nil {
			return fmt.Errorf("failed to create directories for %q: %w", rel, err)
		}
		src, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open the source file %q: %w", path, err)
		}
		defer src.Close()
		dst, err := wt.Filesystem.Create(rel)
		if err != nil {
			return fmt.Errorf("failed to create the destination file %q: %w", rel, err)
		}
		defer dst.Close()
		if _, err := io.Copy(dst, src); err != nil {
			return fmt.Errorf("failed to copy file contents %q -> %q: %w", path, rel, err)
		}
		wt.Add(rel)
		return nil
	}); err != nil {
		t.Fatalf("Failed populating Git repository worktree: %v", err)
	}

	sig := object.Signature{
		Name:  "Porch Unit Test",
		Email: "porch-unit-test@kpt.dev",
		When:  time.Now(),
	}

	hash, err := wt.Commit("Initial Commit", &gogit.CommitOptions{
		All:       true,
		Author:    &sig,
		Committer: &sig,
	})
	if err != nil {
		t.Fatalf("Failed creating initial commit: %v", err)
	}

	main := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/main"), hash)
	if err := repo.Storer.SetReference(main); err != nil {
		t.Fatalf("Failed to set refs/heads/main to commit sha %s: %v", hash, err)
	}
	head := plumbing.NewSymbolicReference(plumbing.HEAD, "refs/heads/main")
	if err := repo.Storer.SetReference(head); err != nil {
		t.Fatalf("Failed to set HEAD to refs/heads/main: %v", err)
	}

	_ = repo.Storer.RemoveReference(plumbing.Master)

	return repo
}

func startGitServer(t *testing.T, repo *git.Repo, opts ...git.GitServerOption) string {
	key := "default"
	repos := git.NewStaticRepos()
	if err := repos.Add(key, repo); err != nil {
		t.Fatalf("repos.Add failed: %v", err)
	}

	server, err := git.NewGitServer(repos)
	if err != nil {
		t.Fatalf("Failed to create git server: %v", err)
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	addressChannel := make(chan net.Addr)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := server.ListenAndServe(ctx, "127.0.0.1:0", addressChannel)
		if err != nil {
			if err == http.ErrServerClosed {
				t.Log("Git server shut down successfully")
			} else {
				t.Errorf("Git server exited with error: %v", err)
			}
		}
	}()

	// Wait for server to start up
	address, ok := <-addressChannel
	if !ok {
		t.Fatalf("Server failed to start")
		return ""
	}

	return fmt.Sprintf("http://%s/%s", address, key)
}

// TODO(mortent): See if we can restruture the packages to
// avoid having to create separate implementations of the auth
// interfaces here.
type credentialResolver struct {
	username, password string
}

type credential struct {
	username, password string
}

func (c *credential) Valid() bool {
	return true
}

func (c *credential) ToAuthMethod() transport.AuthMethod {
	return &githttp.BasicAuth{
		Username: c.username,
		Password: c.password,
	}
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = letters[rand.Intn(len(letters))]
	}
	return string(result)
}

func randomCredentials() *credentialResolver {
	return &credentialResolver{
		username: randomString(30),
		password: randomString(30),
	}
}

func (r *credentialResolver) ResolveCredential(ctx context.Context, namespace, name string) (repository.Credential, error) {
	return &credential{
		username: r.username,
		password: r.password,
	}, nil
}

func TestCloneGitBasicAuth(t *testing.T) {
	testdata, err := filepath.Abs(filepath.Join(".", "testdata", "clone"))
	if err != nil {
		t.Fatalf("Failed to find testdata: %v", err)
	}

	auth := randomCredentials()
	gogitRepo := createRepoWithContents(t, testdata)

	repo, err := git.NewRepo(gogitRepo, git.WithBasicAuth(auth.username, auth.password))
	if err != nil {
		t.Fatalf("NewRepo failed: %v", err)
	}

	addr := startGitServer(t, repo)

	cpm := clonePackageMutation{
		task: &v1alpha1.Task{
			Type: "clone",
			Clone: &v1alpha1.PackageCloneTaskSpec{
				Upstream: v1alpha1.UpstreamPackage{
					Type: "git",
					Git: &v1alpha1.GitPackage{
						Repo:      addr,
						Ref:       "main",
						Directory: "configmap",
						SecretRef: v1alpha1.SecretRef{
							Name: "git-credentials",
						},
					},
				},
			},
		},
		namespace: "test-namespace",
		name:      "test-configmap",
		credentialResolver: &credentialResolver{
			username: "",
			password: "",
		},
	}

	_, _, err = cpm.Apply(context.Background(), repository.PackageResources{})
	if err == nil {
		t.Errorf("Expected error (unauthorized); got none")
	}

	cpm.credentialResolver = auth

	r, _, err := cpm.Apply(context.Background(), repository.PackageResources{})
	if err != nil {
		t.Errorf("task apply failed: %v", err)
	}

	t.Logf("%v", r)
}
