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

// Package update contains libraries for updating packages.
package update

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/util/fetch"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	"github.com/GoogleContainerTools/kpt/internal/util/stack"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

type UpdateOptions struct {
	RelPackagePath string

	LocalPath string

	OriginPath string

	UpdatedPath string

	IsRoot bool

	// DryRun configures AlphaGitPatch to print a patch rather
	// than apply it
	DryRun bool

	// Verbose configures updaters to write verbose output
	Verbose bool

	// SimpleMessage is used for testing so commit messages in patches
	// don't contain the names of generated paths
	SimpleMessage bool

	Output io.Writer
}

// Updater updates a local package
type Updater interface {
	Update(options UpdateOptions) error
}

var strategies = map[kptfilev1alpha2.UpdateStrategyType]func() Updater{
	kptfilev1alpha2.FastForward:        func() Updater { return FastForwardUpdater{} },
	kptfilev1alpha2.ForceDeleteReplace: func() Updater { return ReplaceUpdater{} },
	kptfilev1alpha2.ResourceMerge:      func() Updater { return ResourceMergeUpdater{} },
}

var Strategies = []kptfilev1alpha2.UpdateStrategyType{
	kptfilev1alpha2.FastForward,
	kptfilev1alpha2.ForceDeleteReplace,
	kptfilev1alpha2.ResourceMerge,
}

func StrategiesAsStrings() []string {
	var strs []string
	for _, s := range Strategies {
		strs = append(strs, string(s))
	}
	return strs
}

// Command updates the contents of a local package to a different version.
type Command struct {
	// FullPackagePath is the absolute path to the local package
	FullPackagePath string

	// Ref is the ref to update to
	Ref string

	// Repo is the repo to update to
	Repo string

	// Strategy is the update strategy to use
	Strategy kptfilev1alpha2.UpdateStrategyType

	// DryRun if set will print the patch instead of applying it
	DryRun bool

	// Verbose if set will print verbose information about the commands being run
	Verbose bool

	// SimpleMessage if set will create simple git commit messages that omit values
	// generated for tests
	SimpleMessage bool

	// Output is where dry-run information is written
	Output io.Writer
}

// Run runs the Command.
func (u Command) Run() error {
	if u.Output == nil {
		u.Output = os.Stdout
	}

	// require package is checked into git before trying to update it
	g := gitutil.NewLocalGitRunner(u.FullPackagePath)
	if err := g.Run("status", "-s"); err != nil {
		return errors.Errorf(
			"kpt packages must be checked into a git repo before they are updated: %v", err)
	}
	if strings.TrimSpace(g.Stdout.String()) != "" {
		return errors.Errorf("must commit package %s to git before attempting to update",
			u.FullPackagePath)
	}

	rootKf, err := kptfileutil.ReadFileStrict(u.FullPackagePath)
	if err != nil {
		return errors.Errorf("unable to read package Kptfile: %v", err)
	}

	if rootKf.Upstream == nil || rootKf.Upstream.Git == nil {
		return errors.Errorf("kpt package must have an upstream reference")
	}
	if u.Repo != "" {
		rootKf.Upstream.Git.Repo = u.Repo
	}
	if u.Ref != "" {
		rootKf.Upstream.Git.Ref = u.Ref
	}
	if u.Strategy != "" {
		rootKf.Upstream.UpdateStrategy = u.Strategy
	}
	err = kptfileutil.WriteFile(u.FullPackagePath, rootKf)
	if err != nil {
		return err
	}

	// Use stack to keep track of paths with a Kptfile that might contain
	// information about remote subpackages.
	s := stack.New()
	s.Push(u.FullPackagePath)

	for s.Len() > 0 {
		p := s.Pop()

		if err := u.updatePackage(p); err != nil {
			return err
		}

		paths, err := pkgutil.FindAllDirectSubpackages(p)
		if err != nil {
			return err
		}
		for _, p := range paths {
			s.Push(p)
		}
	}
	return nil
}

func (u Command) updatePackage(p string) error {
	kf, err := kptfileutil.ReadFile(p)
	if err != nil {
		return err
	}

	if kf.Upstream == nil || kf.Upstream.Git == nil {
		return nil
	}

	g := kf.Upstream.Git
	gLock := kf.UpstreamLock.GitLock
	original := &git.RepoSpec{OrgRepo: gLock.Repo, Path: gLock.Directory, Ref: gLock.Commit}
	if err := fetch.ClonerUsingGitExec(original); err != nil {
		return errors.Errorf("failed to clone git repo: original source: %v", err)
	}
	defer os.RemoveAll(original.AbsPath())

	updated := &git.RepoSpec{OrgRepo: g.Repo, Path: g.Directory, Ref: g.Ref}
	if err := fetch.ClonerUsingGitExec(updated); err != nil {
		return errors.Errorf("failed to clone git repo: updated source: %v", err)
	}
	defer os.RemoveAll(updated.AbsPath())

	s := stack.New()
	s.Push(".")

	for s.Len() > 0 {
		relPath := s.Pop()
		isRoot := false
		if relPath == "." {
			isRoot = true
		}

		if !isRoot {
			updatedExists, err := pkgExists(filepath.Join(updated.AbsPath(), relPath))
			if err != nil {
				return err
			}

			originalExists, err := pkgExists(filepath.Join(original.AbsPath(), relPath))
			if err != nil {
				return err
			}

			switch {
			case !originalExists && !updatedExists:
				continue
			case originalExists && !updatedExists:
				if err := os.RemoveAll(p); err != nil {
					return err
				}
				continue
			case !originalExists && updatedExists:
				return fmt.Errorf("package added in both local and upstream")
			default:
			}

			updatedFetched, err := pkgFetched(filepath.Join(updated.AbsPath(), relPath))
			if err != nil {
				return err
			}
			originalFetched, err := pkgFetched(filepath.Join(original.AbsPath(), relPath))
			if err != nil {
				return err
			}

			if !originalFetched || !updatedFetched {
				err := kptfileutil.MergeAndUpdateLocal(
					filepath.Join(p, relPath),
					filepath.Join(updated.AbsPath(), relPath),
					filepath.Join(original.AbsPath(), relPath))
				if err != nil {
					return err
				}
				continue
			}
		}

		pkgKf, err := kptfileutil.ReadFile(filepath.Join(p, relPath))
		if err != nil {
			return err
		}
		updater, found := strategies[pkgKf.Upstream.UpdateStrategy]
		if !found {
			return errors.Errorf("unrecognized update strategy %s", u.Strategy)
		}
		if err := updater().Update(UpdateOptions{
			RelPackagePath: relPath,
			LocalPath:      p,
			UpdatedPath:    updated.AbsPath(),
			OriginPath:     original.AbsPath(),
			IsRoot:         isRoot,
			DryRun:         u.DryRun,
			Verbose:        u.Verbose,
			SimpleMessage:  u.SimpleMessage,
			Output:         u.Output,
		}); err != nil {
			return err
		}

		paths, err := pkgutil.FindRemoteDirectSubpackages(filepath.Join(p, relPath))
		if err != nil {
			return err
		}
		for _, path := range paths {
			rel, err := filepath.Rel(p, path)
			if err != nil {
				return err
			}
			s.Push(rel)
		}
	}
	return fetch.UpsertKptfile(p, updated)
}

func pkgExists(path string) (bool, error) {
	_, err := os.Stat(filepath.Join(path, kptfilev1alpha2.KptFileName))
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return !os.IsNotExist(err), nil
}

func pkgFetched(path string) (bool, error) {
	kf, err := kptfileutil.ReadFile(path)
	if err != nil {
		return false, err
	}
	return kf.UpstreamLock != nil, nil
}
