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
	"io"
	"os"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/util/setters"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

type UpdateOptions struct {
	// KptFile is the current local package KptFile
	KptFile kptfile.KptFile

	// ToRef is the ref to update to
	ToRef string

	// ToRepo is the repo to use for updating
	ToRepo string

	// PackagePath is the relative path to the local package
	PackagePath string

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

	// Perform setters automatically based on environment
	AutoSet bool
}

// Updater updates a local package
type Updater interface {
	Update(options UpdateOptions) error
}

var strategies = map[StrategyType]func() Updater{
	AlphaGitPatch:      func() Updater { return GitPatchUpdater{} },
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

	// AlphaGitPatch will merge upstream changes using `git format-patch` and `git am`.
	AlphaGitPatch StrategyType = "alpha-git-patch"

	KResourceMerge StrategyType = "resource-merge"

	// Default defaults to the recommended strategy, which is FailOnChanges.
	// The recommended strategy may change as new strategies are introduced.
	Default StrategyType = ""
)

var Strategies = []string{
	string(FastForward), string(ForceDeleteReplace), string(AlphaGitPatch), string(KResourceMerge),
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

	// Perform setters automatically based on environment
	AutoSet bool
}

// Run runs the Command.
func (u Command) Run() error {
	if u.Output == nil {
		u.Output = os.Stdout
	}

	kptfile, err := kptfileutil.ReadFileStrict(u.Path)
	if err != nil {
		return errors.Errorf("unable to read package Kptfile: %v", err)
	}

	// default arguments
	if u.Repo == "" {
		u.Repo = kptfile.Upstream.Git.Repo
	}
	if u.Ref == "" {
		u.Ref = kptfile.Upstream.Git.Ref
	}

	// require package is checked into git before trying to update it
	g := gitutil.NewLocalGitRunner("./")
	if err := g.Run("status", "-s", u.Path); err != nil {
		return errors.Errorf(
			"kpt packages must be checked into a git repo before they are updated: %v", err)
	}
	if strings.TrimSpace(g.Stdout.String()) != "" {
		return errors.Errorf("must commit package %s to git before attempting to update",
			u.Path)
	}

	// update
	updater, found := strategies[u.Strategy]
	if !found {
		return errors.Errorf("unrecognized update strategy %s", u.Strategy)
	}
	err = updater().Update(UpdateOptions{
		KptFile:        kptfile,
		ToRef:          u.Ref,
		ToRepo:         u.Repo,
		PackagePath:    u.Path,
		AbsPackagePath: u.FullPackagePath,
		DryRun:         u.DryRun,
		Verbose:        u.Verbose,
		SimpleMessage:  u.SimpleMessage,
		Output:         u.Output,
		AutoSet:        u.AutoSet,
	})

	if err != nil {
		return err
	}

	// perform auto-setters after the package is updated
	a := setters.AutoSet{
		Writer:      u.Output,
		PackagePath: u.Path,
	}
	return a.PerformAutoSetters()
}
