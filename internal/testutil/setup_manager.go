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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type TestSetupManager struct {
	T *testing.T

	// GetRef is the git ref to fetch
	GetRef string

	// GetSubDirectory is the repo subdirectory containing the package
	GetSubDirectory string

	// UpstreamInit are made before fetching the repo
	UpstreamInit []Content

	// UpstreamChanges are upstream content changes made after cloning the repo
	UpstreamChanges []Content

	// RefReposChanges are content for any other repos that will be used as
	// remote subpackages.
	RefReposChanges map[string][]Content

	LocalChanges []Content

	UpstreamRepo *TestGitRepo

	RefRepos map[string]*TestGitRepo

	LocalWorkspace *TestWorkspace

	RepoPaths map[string]string

	cleanTestRepo func()
	cacheDir      string
	targetDir     string
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
func (g *TestSetupManager) Init(content Content) bool {
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

	// Set up any repos that will be used as remote subpackages.
	refRepos, err := SetupRepos(g.T, g.RefReposChanges)
	if !assert.NoError(g.T, err) {
		return false
	}
	g.RefRepos = refRepos

	// Create the mapping from repo name to path.
	g.RepoPaths = make(map[string]string)
	for name, tgr := range refRepos {
		g.RepoPaths[name] = tgr.RepoDirectory
	}

	// Setup a "remote" source repo, and a "local" destination repo
	g.UpstreamRepo, g.LocalWorkspace, g.cleanTestRepo = SetupDefaultRepoAndWorkspace(g.T, content, g.RepoPaths)
	if g.GetSubDirectory == "/" {
		g.targetDir = filepath.Base(g.UpstreamRepo.RepoName)
	} else {
		g.targetDir = filepath.Base(g.GetSubDirectory)
	}
	g.LocalWorkspace.PackageDir = g.targetDir

	// Update the upstream repo with the init content.
	if err := UpdateGitDir(g.T, g.UpstreamRepo, g.UpstreamInit, g.RepoPaths); err != nil {
		return false
	}

	// Get the content from the upstream repo into the local workspace.
	if !assert.NoError(g.T, get.Command{
		Destination: filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.targetDir),
		Git: &kptfilev1alpha2.Git{
			Repo:      g.UpstreamRepo.RepoDirectory,
			Ref:       g.GetRef,
			Directory: g.GetSubDirectory,
		}}.Run()) {
		return false
	}
	localGit := gitutil.NewLocalGitRunner(g.LocalWorkspace.WorkspaceDirectory)
	if !assert.NoError(g.T, localGit.Run("add", ".")) {
		return false
	}
	if !assert.NoError(g.T, localGit.Run("commit", "-m", "add files")) {
		return false
	}

	// Modify source repository state after fetching it
	if err := UpdateGitDir(g.T, g.UpstreamRepo, g.UpstreamChanges, g.RepoPaths); err != nil {
		return false
	}

	// Modify local workspace after initial fetch.
	if err := UpdateGitDir(g.T, g.LocalWorkspace, g.LocalChanges, g.RepoPaths); err != nil {
		return false
	}

	// Modify other repos after initial fetch.
	if err := UpdateRefRepos(g.T, refRepos, g.RefReposChanges, g.RepoPaths); err != nil {
		return false
	}

	return true
}

type GitDirectory interface {
	CheckoutBranch(branch string, create bool) error
	ReplaceData(data string) error
	Commit(message string) error
	Tag(tagName string) error
}

func UpdateGitDir(t *testing.T, gitDir GitDirectory, changes []Content, repoPaths map[string]string) error {
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
			pkgData = pkgbuilder.ExpandPkg(t, content.Pkg, repoPaths)
		} else {
			pkgData = content.Data
		}

		err := gitDir.ReplaceData(pkgData)
		if !assert.NoError(t, err) {
			return err
		}

		err = gitDir.Commit(content.Message)
		if !assert.NoError(t, err) {
			return err
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
				Repo:      g.UpstreamRepo.RepoDirectory,
				Ref:       ref,
			},
			UpdateStrategy: strategy,
		},
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: "git",
			GitLock: &kptfilev1alpha2.GitLock{
				Directory: g.GetSubDirectory,
				Repo:      g.UpstreamRepo.RepoDirectory,
				Ref:       ref,
				Commit:    commit,
			},
		},
	}

	return g.UpstreamRepo.AssertKptfile(
		g.T, filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.targetDir), expectedKptfile)
}

func (g *TestSetupManager) AssertLocalDataEquals(path string) bool {
	var sourceDir string
	if filepath.IsAbs(path) {
		sourceDir = path
	} else {
		sourceDir = filepath.Join(g.UpstreamRepo.DatasetDirectory, path)
	}
	destDir := filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.targetDir)
	return g.UpstreamRepo.AssertEqual(g.T, sourceDir, destDir)
}

func (g *TestSetupManager) Clean() {
	g.cleanTestRepo()
	os.RemoveAll(g.cacheDir)
}
