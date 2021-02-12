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

// Package get contains libraries for fetching packages.
package get

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/util/fetch"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	"github.com/GoogleContainerTools/kpt/internal/util/stack"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/pathutil"
)

// Command fetches a package from a git repository, copies it to a local
// directory, and expands any remote subpackages.
type Command struct {
	// Git contains information about the git repo to fetch
	kptfilev1alpha2.GitLock

	// Destination is the output directory to clone the package to.  Defaults to the name of the package --
	// either the base repo name, or the base subdirectory name.
	Destination string

	// Name is the name to give the package.  Defaults to the destination.
	Name string

	// Remove directory before copying to it.
	Clean bool
}

// Run runs the Command.
func (c Command) Run() error {
	revertFunc, err := c.updateParentKptfile()
	if err != nil {
		return err
	}

	r := &git.RepoSpec{OrgRepo: c.Repo, Path: c.Directory, Ref: c.Ref}
	err = (&fetch.Command{
		RepoSpec:    r,
		Destination: c.Destination,
		Name:        c.Name,
		Clean:       c.Clean,
	}).Run()
	if err != nil {
		// Ignore the error here. If this happens, it just means that
		// we weren't able to roll back the change to the parent Kptfile.
		_ = revertFunc()
		return err
	}

	if err = c.fetchRemoteSubpackages(); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

// updateParentKptfile searches the parent folders of a Kptfile. If it finds
// a Kptfile, it means the package should be registered as a subpackage of the
// parent. It adds the new package to the parent. The function returns a function
// that makes it possible to revert the change if fetching the package fails.
func (c Command) updateParentKptfile() (func() error, error) {
	return pkgutil.UpdateParentKptfile(c.Destination, func(parentPath string, kf kptfilev1alpha2.KptFile) (kptfilev1alpha2.KptFile, error) {
		for _, subPkg := range kf.Subpackages {
			absPath := filepath.Join(parentPath, subPkg.LocalDir)
			if absPath == c.Destination {
				return kptfilev1alpha2.KptFile{}, fmt.Errorf("subpackage with localDir %q already exist", subPkg.LocalDir)
			}
		}

		relPkgPath, err := filepath.Rel(parentPath, c.Destination)
		if err != nil {
			return kptfilev1alpha2.KptFile{}, err
		}

		kf.Subpackages = append(kf.Subpackages, kptfilev1alpha2.Subpackage{
			LocalDir: relPkgPath,
			Upstream: &kptfilev1alpha2.Upstream{
				Git: &kptfilev1alpha2.Git{
					Repo:      c.Repo,
					Directory: c.Directory,
					Ref:       c.Ref,
				},
				UpdateStrategy: "resource-merge",
			},
		})
		return kf, nil
	})
}

// fetchRemoteSubpackages goes through the root package and its subpackages
// and fetches any remote subpackages referenced. It will also handle situations
// where a remote subpackage references other remote subpackages.
func (c Command) fetchRemoteSubpackages() error {
	// Create a stack to keep track of all Kptfiles that needs to be checked
	// for remote subpackages.
	s := stack.New()

	paths, err := pathutil.DirsWithFile(c.Destination, kptfilev1alpha2.KptFileName, true)
	if err != nil {
		return err
	}
	for _, p := range paths {
		s.Push(p)
	}

	for s.Len() > 0 {
		p := s.Pop()
		kf, err := kptfileutil.ReadFile(p)
		if err != nil {
			return err
		}

		remoteSubPkgDirs := make(map[string]bool)
		for i := range kf.Subpackages {
			sp := kf.Subpackages[i]

			if _, found := remoteSubPkgDirs[sp.LocalDir]; found {
				return fmt.Errorf("multiple remote subpackages with localDir %q", sp.LocalDir)
			}
			remoteSubPkgDirs[sp.LocalDir] = true

			gitInfo := sp.Upstream.Git
			localPath := filepath.Join(p, sp.LocalDir)

			_, err = os.Stat(localPath)
			// If we get an error and it is something different than that the
			// directory doesn't exist, we just return the error.
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			// Check if the folder already exist by checking if err is nil. Due
			// to the check above, err here can only be IsNotExist or nil. So
			// if err is nil it means the folder already exists.
			// If it does, we return an error with a specific error message.
			if err == nil {
				return fmt.Errorf("local subpackage in directory %q already exists. Either"+
					"rename the local subpackage or use a different directory for the remote subpackage", sp.LocalDir)
			}

			r := &git.RepoSpec{OrgRepo: gitInfo.Repo, Path: gitInfo.Directory, Ref: gitInfo.Ref}
			err := (&fetch.Command{
				RepoSpec:    r,
				Destination: localPath,
				Name:        sp.LocalDir,
				Clean:       false,
			}).Run()
			if err != nil {
				return err
			}

			subPaths, err := pathutil.DirsWithFile(localPath, kptfilev1alpha2.KptFileName, true)
			if err != nil {
				return err
			}
			for _, subp := range subPaths {
				if subp == p {
					continue
				}
				s.Push(subp)
			}
		}
	}
	return nil
}
