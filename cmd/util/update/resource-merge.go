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

package update

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"lib.kpt.dev/kio/filters"
	"lib.kpt.dev/kptfile"
	"lib.kpt.dev/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/v3/pkg/git"
)

// ResourceMergeUpdater updates a package by fetching the original and updated source
// packages, and performing a 3-way merge of the Resources.
type ResourceMergeUpdater struct{}

func (u ResourceMergeUpdater) Update(options UpdateOptions) error {
	g := options.KptFile.Upstream.Git
	g.Ref = options.ToRef
	g.Repo = options.ToRepo

	// get the original repo
	original := &git.RepoSpec{OrgRepo: g.Repo, Path: g.Directory, Ref: g.Commit}
	if err := git.ClonerUsingGitExec(original); err != nil {
		return fmt.Errorf("failed to clone git repo: original source: %v", err)
	}
	defer os.RemoveAll(original.AbsPath())

	// get the updated repo
	updated := &git.RepoSpec{OrgRepo: options.ToRepo, Path: g.Directory, Ref: options.ToRef}
	if err := git.ClonerUsingGitExec(updated); err != nil {
		return fmt.Errorf("failed to clone git repo: updated source: %v", err)
	}
	defer os.RemoveAll(updated.AbsPath())

	// get the Kptfile to write after the merge
	kf, err := u.updatedKptfile(updated.AbsPath(), options)
	if err != nil {
		return err
	}

	// merge the Resources: original + updated + dest => dest
	err = filters.Merge3{
		OriginalPath: original.AbsPath(),
		UpdatedPath:  updated.AbsPath(),
		DestPath:     options.PackagePath,
	}.Merge()
	if err != nil {
		return err
	}

	return kptfileutil.WriteFile(options.PackagePath, kf)
}

// updatedKptfile returns a Kptfile to replace the existing local Kptfile as part of the udpate
func (u ResourceMergeUpdater) updatedKptfile(updatedPath string, options UpdateOptions) (
	kptfile.KptFile, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = updatedPath
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return kptfile.KptFile{}, err
	}
	commit := strings.TrimSpace(string(b))
	kf, err := kptfileutil.ReadFile(updatedPath)
	if err != nil {
		kf, err = kptfileutil.ReadFile(options.PackagePath)
		if err != nil {
			return kptfile.KptFile{}, err
		}
	}

	// local package controls the upstream field
	kf.Upstream = options.KptFile.Upstream
	kf.Upstream.Git.Commit = commit
	kf.Upstream.Git.Ref = options.ToRef
	kf.Upstream.Git.Repo = options.ToRepo
	return kf, err
}
