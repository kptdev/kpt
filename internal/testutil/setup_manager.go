// Copyright 2019 Google LLC
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

package testutil

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	Upstream = "upstream"
	Local    = "local"
)

type TestSetupManager struct {
	T *testing.T

	// GetRef is the git ref to fetch
	GetRef string

	// GetSubDirectory is the repo subdirectory containing the package
	GetSubDirectory string

	// ReposChanges are content for any repos.
	ReposChanges map[string][]Content

	LocalChanges []Content

	Repos map[string]*TestGitRepo

	LocalWorkspace *TestWorkspace

	cleanTestRepos func()
	cacheDir       string
	targetDir      string
}

type Content struct {
	CreateBranch bool
	Branch       string
	Data         string
	Pkg          *pkgbuilder.RootPkg
	Tag          string
	Message      string
}

// Init initializes test data
// - Setup a new upstream repo in a tmp directory
// - Set the initial upstream content to Dataset1
// - Setup a new cache location for git repos and update the environment variable
// - Setup fetch the upstream package to a local package
// - Verify the local package contains the upstream content
func (g *TestSetupManager) Init() bool {
	// Default optional values
	if g.GetRef == "" {
		g.GetRef = "master"
	}
	if g.GetSubDirectory == "" {
		g.GetSubDirectory = "/"
	}

	// Configure the cache location for cloning repos
	cacheDir, err := ioutil.TempDir("", "kpt-test-cache-repos-")
	if !assert.NoError(g.T, err) {
		return false
	}
	g.cacheDir = cacheDir
	os.Setenv(gitutil.RepoCacheDirEnv, g.cacheDir)

	g.Repos, g.LocalWorkspace, g.cleanTestRepos = SetupReposAndWorkspace(g.T, g.ReposChanges)
	if g.GetSubDirectory == "/" {
		g.targetDir = filepath.Base(g.Repos[Upstream].RepoName)
	} else {
		g.targetDir = filepath.Base(g.GetSubDirectory)
	}
	g.LocalWorkspace.PackageDir = g.targetDir

	// Get the content from the upstream repo into the local workspace.
	if !assert.NoError(g.T, get.Command{
		Destination: filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.targetDir),
		Git: &kptfilev1alpha2.Git{
			Repo:      g.Repos[Upstream].RepoDirectory,
			Ref:       g.GetRef,
			Directory: g.GetSubDirectory,
		}}.Run(fake.CtxWithNilPrinter())) {
		return false
	}
	localGit, err := gitutil.NewLocalGitRunner(g.LocalWorkspace.WorkspaceDirectory)
	if !assert.NoError(g.T, err) {
		return false
	}
	_, err = localGit.Run(context.Background(), "add", ".")
	if !assert.NoError(g.T, err) {
		return false
	}
	_, err = localGit.Run(context.Background(), "commit", "-m", "add files")
	if !assert.NoError(g.T, err) {
		return false
	}

	// Modify other repos after initial fetch.
	if err := UpdateRepos(g.T, g.Repos, g.ReposChanges); err != nil {
		return false
	}

	// Modify local workspace after initial fetch.
	if err := UpdateGitDir(g.T, Local, g.LocalWorkspace, g.LocalChanges, g.Repos); err != nil {
		return false
	}

	return true
}

type GitDirectory interface {
	CheckoutBranch(branch string, create bool) error
	ReplaceData(data string) error
	Commit(message string) (string, error)
	Tag(tagName string) error
}

func UpdateGitDir(t *testing.T, name string, gitDir GitDirectory, changes []Content, repos map[string]*TestGitRepo) error {
	for _, content := range changes {
		if content.Message == "" {
			content.Message = "initializing data"
		}
		if len(content.Branch) > 0 {
			err := gitDir.CheckoutBranch(content.Branch, content.CreateBranch)
			if !assert.NoError(t, err) {
				return err
			}
		}

		var pkgData string
		if content.Pkg != nil {
			pkgData = content.Pkg.ExpandPkg(t, ToReposInfo(repos))
		} else {
			pkgData = content.Data
		}

		err := gitDir.ReplaceData(pkgData)
		if !assert.NoError(t, err) {
			return err
		}

		sha, err := gitDir.Commit(content.Message)
		if !assert.NoError(t, err) {
			return err
		}

		// Update the list of commit shas for the repo.
		if r, found := repos[name]; found {
			r.Commits = append(r.Commits, sha)
		}

		if len(content.Tag) > 0 {
			err = gitDir.Tag(content.Tag)
			if !assert.NoError(t, err) {
				return err
			}
		}
	}
	return nil
}

func (g *TestSetupManager) AssertKptfile(name, commit, ref string, strategy kptfilev1alpha2.UpdateStrategyType) bool {
	expectedKptfile := kptfilev1alpha2.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: name,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1alpha2.TypeMeta.APIVersion,
				Kind:       kptfilev1alpha2.TypeMeta.Kind},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: "git",
			Git: &kptfilev1alpha2.Git{
				Directory: g.GetSubDirectory,
				Repo:      g.Repos[Upstream].RepoDirectory,
				Ref:       ref,
			},
			UpdateStrategy: strategy,
		},
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: "git",
			Git: &kptfilev1alpha2.GitLock{
				Directory: g.GetSubDirectory,
				Repo:      g.Repos[Upstream].RepoDirectory,
				Ref:       ref,
				Commit:    commit,
			},
		},
	}

	return g.Repos[Upstream].AssertKptfile(
		g.T, filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.targetDir), expectedKptfile)
}

func (g *TestSetupManager) AssertLocalDataEquals(path string) bool {
	var sourceDir string
	if filepath.IsAbs(path) {
		sourceDir = path
	} else {
		sourceDir = filepath.Join(g.Repos[Upstream].DatasetDirectory, path)
	}
	destDir := filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.targetDir)
	return g.Repos[Upstream].AssertEqual(g.T, sourceDir, destDir)
}

func (g *TestSetupManager) Clean() {
	g.cleanTestRepos()
	os.RemoveAll(g.cacheDir)
}
