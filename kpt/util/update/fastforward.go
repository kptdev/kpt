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

	"kpt.dev/kpt/util/get"
	"lib.kpt.dev/kptfile"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/v3/pkg/git"
)

// Updater updates a package to a new upstream version.
//
// If the package at pkgPath differs from the upstream ref it was fetch from, then Update will
// fail without making any changes.
type FastForwardUpdater struct{}

var kptfileSet = func() sets.String {
	s := sets.String{}
	s.Insert(kptfile.KptFileName)
	return s
}()

func (u FastForwardUpdater) Update(options UpdateOptions) error {
	g := options.KptFile.Upstream.Git
	g.Ref = options.ToRef
	g.Repo = options.ToRepo
	if err := errorIfChanged(g, options.PackagePath); err != nil {
		return err
	}

	// refetch the package
	return get.Command{Destination: options.PackagePath, Clean: true, Git: g}.Run()
}

// errorIfChanged returns an error if the package at pkgPath has changed from the upstream
// source referenced by g.
func errorIfChanged(g kptfile.Git, pkgPath string) error {
	original := &git.RepoSpec{
		OrgRepo: g.Repo,
		Path:    g.Directory,
		Ref:     g.Commit,
	}
	err := get.ClonerUsingGitExec(original)
	if err != nil {
		return errors.Errorf("failed cloning git repo: %v", err)
	}
	defer os.RemoveAll(original.AbsPath())
	diff, err := copyutil.Diff(original.AbsPath(), pkgPath)
	if err != nil {
		return errors.Errorf("failed to compare local package to original source: %v", err)
	}

	diff = diff.Difference(kptfileSet)
	if diff.Len() > 0 {
		return DiffError(fmt.Sprintf(
			"local package files have been modified: %v.\n  use a differnt update --strategy.",
			diff.List()))
	}
	return nil
}

// DiffError is returned if the local package and upstream package contents do not match.
type DiffError string

func (d DiffError) Error() string {
	return string(d)
}
