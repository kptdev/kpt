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
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	"github.com/GoogleContainerTools/kpt/internal/util/stack"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

type UpdateOptions struct {
	// KptFile is the current local package KptFile
	KptFile kptfilev1alpha2.KptFile

	// ToRef is the ref to update to
	ToRef string

	// ToRepo is the repo to use for updating
	ToRepo string

	// AbsPackagePath is the absolute path to the local package
	AbsPackagePath string

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

var strategies = map[StrategyType]func() Updater{
	Default:            func() Updater { return ResourceMergeUpdater{} },
	FastForward:        func() Updater { return FastForwardUpdater{} },
	ForceDeleteReplace: func() Updater { return ReplaceUpdater{} },
	KResourceMerge:     func() Updater { return ResourceMergeUpdater{} },
}

// StrategyType controls the update strategy to use when the local package
// has been modifed from its original remote source.
type StrategyType string

const (
	// FastForward will fail the package update if the local
	// package contents do not match the contents for the remote
	// repository at the commit it was fetched from
	FastForward StrategyType = "fast-forward"

	// ForceDeleteReplace will delete the existing local package
	// and replace the contents with a new version fetched from
	// a remote repository
	ForceDeleteReplace StrategyType = "force-delete-replace"

	KResourceMerge StrategyType = "resource-merge"

	// Default defaults to the recommended strategy, which is FailOnChanges.
	// The recommended strategy may change as new strategies are introduced.
	Default StrategyType = ""
)

var Strategies = []string{
	string(FastForward), string(ForceDeleteReplace), string(KResourceMerge),
}

// Command updates the contents of a local package to a different version.
type Command struct {
	// Path is the filepath to the local package
	Path string

	// FullPackagePath is the absolute path to the local package
	FullPackagePath string

	// Ref is the ref to update to
	Ref string

	// Repo is the repo to update to
	Repo string

	// Strategy is the update strategy to use
	Strategy StrategyType

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

	rootKf, err := kptfileutil.ReadFileStrict(u.FullPackagePath)
	if err != nil {
		return errors.Errorf("unable to read package Kptfile: %v", err)
	}

	// default arguments
	if rootKf.UpstreamLock != nil && rootKf.UpstreamLock.GitLock != nil {
		if u.Repo == "" {
			u.Repo = rootKf.UpstreamLock.GitLock.Repo
		}
		if u.Ref == "" {
			u.Ref = rootKf.UpstreamLock.GitLock.Ref
		}
	}

	// require package is checked into git before trying to update it
	g := gitutil.NewLocalGitRunner(u.FullPackagePath)
	if err := g.Run("status", "-s"); err != nil {
		return errors.Errorf(
			"kpt packages must be checked into a git repo before they are updated: %v", err)
	}
	if strings.TrimSpace(g.Stdout.String()) != "" {
		return errors.Errorf("must commit package %s to git before attempting to update",
			u.Path)
	}

	// update root package
	updater, found := strategies[u.Strategy]
	if !found {
		return errors.Errorf("unrecognized update strategy %s", u.Strategy)
	}

	revertFunc, err := u.updateParentKptfile()
	if err != nil {
		return err
	}

	if err := updater().Update(UpdateOptions{
		KptFile:        rootKf,
		ToRef:          u.Ref,
		ToRepo:         u.Repo,
		AbsPackagePath: u.FullPackagePath,
		DryRun:         u.DryRun,
		Verbose:        u.Verbose,
		SimpleMessage:  u.SimpleMessage,
		Output:         u.Output,
	}); err != nil {
		_ = revertFunc()
		return err
	}

	// Use stack to keep track of paths with a Kptfile that might contain
	// information about remote subpackages.
	s := stack.New()
	s.Push(u.FullPackagePath)

	for s.Len() > 0 {
		p := s.Pop()

		kf, err := kptfileutil.ReadFile(p)
		if err != nil {
			return err
		}

		for _, sp := range kf.Subpackages {
			spFilePath := filepath.Join(p, sp.LocalDir)

			_, err := os.Stat(spFilePath)
			if err != nil && !os.IsNotExist(err) {
				return err
			}

			if os.IsNotExist(err) {
				if err := (get.Command{
					GitLock: kptfilev1alpha2.GitLock{
						Repo:      sp.Upstream.Git.Repo,
						Ref:       sp.Upstream.Git.Ref,
						Directory: sp.Upstream.Git.Directory,
					},
					Destination: spFilePath,
					Name:        sp.LocalDir,
					Clean:       false,
				}).Run(); err != nil {
					return err
				}
				continue
			}

			spKptfile, err := kptfileutil.ReadFile(spFilePath)
			if err != nil {
				return err
			}

			// If either the repo or the directory of the current local package
			// doesn't match the remote subpackage spec in the Kptfile, it must
			// be a local subpackage.
			if sp.Upstream.Git.Repo != spKptfile.UpstreamLock.GitLock.Repo ||
				sp.Upstream.Git.Directory != spKptfile.UpstreamLock.GitLock.Directory {
				return fmt.Errorf("subpackage already exists in directory %s", sp.LocalDir)
			}

			updater, found := strategies[StrategyType(sp.Upstream.UpdateStrategy)]
			if !found {
				return errors.Errorf("unrecognized update strategy %s", u.Strategy)
			}
			if err := updater().Update(UpdateOptions{
				KptFile:        spKptfile,
				ToRef:          sp.Upstream.Git.Ref,
				ToRepo:         sp.Upstream.Git.Repo,
				AbsPackagePath: spFilePath,
				DryRun:         u.DryRun,
				Verbose:        u.Verbose,
				SimpleMessage:  u.SimpleMessage,
				Output:         u.Output,
			}); err != nil {
				return err
			}
			s.Push(spFilePath)
		}
	}
	return nil
}

// updateParentKptfile searches the parent folders of a Kptfile. If it finds
// a Kptfile, it means the parent Kptfile should be updated with the new
// information about the remote subpackage. The function returns a function
// that makes it possible to revert the change if fetching the package fails.
func (u Command) updateParentKptfile() (func() error, error) {
	return pkgutil.UpdateParentKptfile(u.FullPackagePath, func(parentPath string, kf kptfilev1alpha2.KptFile) (kptfilev1alpha2.KptFile, error) {
		var found bool
		for i := range kf.Subpackages {
			absPath := filepath.Join(parentPath, kf.Subpackages[i].LocalDir)
			if absPath == u.FullPackagePath {
				kf.Subpackages[i].Upstream.Git.Repo = u.Repo
				kf.Subpackages[i].Upstream.Git.Ref = u.Ref
				found = true
				break
			}
		}

		if !found {
			return kptfilev1alpha2.KptFile{}, fmt.Errorf("subpackage at %q not listed in parent Kptfile", u.FullPackagePath)
		}
		return kf, nil
	})
}
