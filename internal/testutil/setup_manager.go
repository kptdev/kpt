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
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
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

	LocalChanges []Content

	UpstreamRepo *TestGitRepo

	LocalWorkspace *TestWorkspace

	cleanTestRepo func()
	cacheDir      string
	targetDir     string
}

type Content struct {
	CreateBranch bool
	Branch       string
	Data         string
	Pkg          *pkgbuilder.Pkg
	Tag          string
	Message      string
}

// Init initializes test data
// - Setup a new upstream repo in a tmp directory
// - Set the initial upstream content to Dataset1
// - Setup a new cache location for git repos and update the environment variable
// - Setup fetch the upstream package to a local package
// - Verify the local package contains the upstream content
func (g *TestSetupManager) Init(dataset string) bool {
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

	// Setup a "remote" source repo, and a "local" destination repo
	g.UpstreamRepo, g.LocalWorkspace, g.cleanTestRepo = SetupDefaultRepoAndWorkspace(g.T, dataset)
	if g.GetSubDirectory == "/" {
		g.targetDir = filepath.Base(g.UpstreamRepo.RepoName)
	} else {
		g.targetDir = filepath.Base(g.GetSubDirectory)
	}
	if !assert.NoError(g.T, os.Chdir(g.UpstreamRepo.RepoDirectory)) {
		return false
	}

	if err := updateGitDir(g.T, g.UpstreamRepo, g.UpstreamInit); err != nil {
		return false
	}

	// Fetch the source repo
	if !assert.NoError(g.T, os.Chdir(g.LocalWorkspace.WorkspaceDirectory)) {
		return false
	}

	if !assert.NoError(g.T, get.Command{
		Destination: g.targetDir,
		Git: kptfile.Git{
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
	if err := updateGitDir(g.T, g.UpstreamRepo, g.UpstreamChanges); err != nil {
		return false
	}

	// Verify the local package has the correct dataset
	if same := g.AssertLocalDataEquals(filepath.Join(dataset, g.GetSubDirectory)); !same {
		return same
	}

	if err := updateGitDir(g.T, g.LocalWorkspace, g.LocalChanges); err != nil {
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

func updateGitDir(t *testing.T, gitDir GitDirectory, changes []Content) error {
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
			pkgData = pkgbuilder.ExpandPkg(t, content.Pkg)
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

func (g *TestSetupManager) AssertKptfile(name, commit, ref string) bool {
	expectedKptfile := kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: name,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
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

func (g *TestSetupManager) AssertKptfileEquals(path, commit, ref string) bool {
	kf, err := kptfileutil.ReadFile(path)
	if !assert.NoError(g.T, err) {
		g.T.FailNow()
	}
	kf.Upstream.Type = kptfile.GitOrigin
	kf.Upstream.Git.Directory = g.GetSubDirectory
	kf.Upstream.Git.Commit = commit
	kf.Upstream.Git.Ref = ref
	kf.Upstream.Git.Repo = g.UpstreamRepo.RepoDirectory
	return g.UpstreamRepo.AssertKptfile(g.T, g.LocalWorkspace.FullPackagePath(), kf)
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

func (g *TestSetupManager) SetLocalData(path string) bool {
	if !assert.NoError(g.T, copyutil.CopyDir(
		filepath.Join(g.UpstreamRepo.DatasetDirectory, path),
		filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.UpstreamRepo.RepoName))) {
		return false
	}
	localGit := gitutil.NewLocalGitRunner(g.LocalWorkspace.WorkspaceDirectory)
	if !assert.NoError(g.T, localGit.Run("add", ".")) {
		return false
	}
	if !assert.NoError(g.T, localGit.Run("commit", "-m", "add files")) {
		return false
	}
	return true
}

func (g *TestSetupManager) Clean() {
	g.cleanTestRepo()
	os.RemoveAll(g.cacheDir)
}
