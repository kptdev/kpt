// Copyright 2019 The kpt Authors
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

// Package diff contains libraries for diffing packages.
package diff

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/addmergecomment"
	"github.com/GoogleContainerTools/kpt/internal/util/fetch"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// Type represents type of diff comparison to be performed.
type Type string

const (
	// TypeLocal shows the changes in local pkg relative to upstream source pkg at original version
	TypeLocal Type = "local"
	// TypeRemote shows changes in the upstream source pkg between original and target version
	TypeRemote Type = "remote"
	// TypeCombined shows changes in local pkg relative to upstream source pkg at target version
	TypeCombined Type = "combined"
	// 3way shows changes in local and remote changes side-by-side
	Type3Way Type = "3way"
)

// A collection of user-readable "source" definitions for diffed packages.
const (
	// localPackageSource represents the local package
	LocalPackageSource string = "local"
	// remotePackageSource represents the remote version of the package
	RemotePackageSource string = "remote"
	// targetRemotePackageSource represents the targeted remote version of a package
	TargetRemotePackageSource string = "target"
)

const (
	exitCodeDiffWarning string = "\nThe selected diff tool (%s) exited with an " +
		"error. It may not support the chosen diff type (%s). To use a different " +
		"diff tool please provide the tool using the --diff-tool flag. \n\nFor " +
		"more information about using kpt's diff command please see the commands " +
		"--help.\n"
)

// String implements Stringer.
func (dt Type) String() string {
	return string(dt)
}

var SupportedDiffTypes = []Type{TypeLocal, TypeRemote, TypeCombined, Type3Way}

func SupportedDiffTypesLabel() string {
	var labels []string
	for _, dt := range SupportedDiffTypes {
		labels = append(labels, dt.String())
	}
	return strings.Join(labels, ", ")
}

// Command shows changes in local package relative to upstream source pkg, changes in
// upstream source package between original and target version etc.
type Command struct {
	// Path to the local package directory
	Path string

	// Ref is the target Ref in the upstream source package to compare against
	Ref string

	// DiffType specifies the type of changes to show
	DiffType Type

	// Difftool refers to diffing commandline tool for showing changes.
	DiffTool string

	// DiffToolOpts refers to the commandline options to for the diffing tool.
	DiffToolOpts string

	// When Debug is true, command will run with verbose logging and will not
	// cleanup the staged packages to assist with debugging.
	Debug bool

	// Output is an io.Writer where command will write the output of the
	// command.
	Output io.Writer

	// PkgDiffer specifies package differ
	PkgDiffer PkgDiffer

	// PkgGetter specifies packaging sourcing adapter
	PkgGetter PkgGetter
}

func (c *Command) Run(ctx context.Context) error {
	c.DefaultValues()

	kptFile, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, c.Path)
	if err != nil {
		return errors.Errorf("package missing Kptfile at '%s': %v", c.Path, err)
	}

	// Return early if upstream is not set
	if kptFile.Upstream == nil || kptFile.Upstream.Git == nil {
		return errors.Errorf("package missing upstream in Kptfile at '%s'", c.Path)
	}

	// Create a staging directory to store all compared packages
	stagingDirectory, err := os.MkdirTemp("", "kpt-")
	if err != nil {
		return errors.Errorf("failed to create stage dir: %v", err)
	}
	defer func() {
		// Cleanup staged content after diff. Ignore cleanup if debugging.
		if !c.Debug {
			defer os.RemoveAll(stagingDirectory)
		}
	}()

	// Stage current package
	// This prevents prepareForDiff from modifying the local package
	localPkgName := NameStagingDirectory(LocalPackageSource,
		kptFile.Upstream.Git.Ref)
	currPkg, err := stageDirectory(stagingDirectory, localPkgName)
	if err != nil {
		return errors.Errorf("failed to create stage dir for current package: %v", err)
	}

	err = pkgutil.CopyPackage(c.Path, currPkg, true, pkg.Local)
	if err != nil {
		return errors.Errorf("failed to stage current package: %v", err)
	}

	// get the upstreamPkg at current version
	upstreamPkgName := NameStagingDirectory(RemotePackageSource,
		kptFile.Upstream.Git.Ref)
	upstreamPkg, err := c.PkgGetter.GetPkg(ctx,
		stagingDirectory,
		upstreamPkgName,
		kptFile.Upstream.Git.Repo,
		kptFile.Upstream.Git.Directory,
		kptFile.Upstream.Git.Ref)
	if err != nil {
		return err
	}

	var upstreamTargetPkg string

	if c.Ref == "" {
		gur, err := gitutil.NewGitUpstreamRepo(ctx, kptFile.UpstreamLock.Git.Repo)
		if err != nil {
			return err
		}
		c.Ref, err = gur.GetDefaultBranch(ctx)
		if err != nil {
			return err
		}
	}

	if c.DiffType == TypeRemote ||
		c.DiffType == TypeCombined ||
		c.DiffType == Type3Way {
		// get the upstream pkg at the target version
		upstreamTargetPkgName := NameStagingDirectory(TargetRemotePackageSource,
			c.Ref)
		upstreamTargetPkg, err = c.PkgGetter.GetPkg(ctx, stagingDirectory,
			upstreamTargetPkgName,
			kptFile.Upstream.Git.Repo,
			kptFile.Upstream.Git.Directory,
			c.Ref)
		if err != nil {
			return err
		}
	}

	if c.Debug {
		fmt.Fprintf(c.Output, "diffing currPkg: %v, upstreamPkg: %v, upstreamTargetPkg: %v \n",
			currPkg, upstreamPkg, upstreamTargetPkg)
	}

	switch c.DiffType {
	case TypeLocal:
		return c.PkgDiffer.Diff(currPkg, upstreamPkg)
	case TypeRemote:
		return c.PkgDiffer.Diff(upstreamPkg, upstreamTargetPkg)
	case TypeCombined:
		return c.PkgDiffer.Diff(currPkg, upstreamTargetPkg)
	case Type3Way:
		return c.PkgDiffer.Diff(currPkg, upstreamPkg, upstreamTargetPkg)
	default:
		return errors.Errorf("unsupported diff type '%s'", c.DiffType)
	}
}

func (c *Command) Validate() error {
	switch c.DiffType {
	case TypeLocal, TypeCombined, TypeRemote, Type3Way:
	default:
		return errors.Errorf("invalid diff-type '%s': supported diff-types are: %s",
			c.DiffType, SupportedDiffTypesLabel())
	}

	path, err := exec.LookPath(c.DiffTool)
	if err != nil {
		return errors.Errorf("diff-tool '%s' not found in the PATH", c.DiffTool)
	}
	c.DiffTool = path
	return nil
}

// DefaultValues sets up the default values for the command.
func (c *Command) DefaultValues() {
	if c.Output == nil {
		c.Output = os.Stdout
	}
	if c.PkgGetter == nil {
		c.PkgGetter = defaultPkgGetter{}
	}
	if c.PkgDiffer == nil {
		c.PkgDiffer = &defaultPkgDiffer{
			DiffType:     c.DiffType,
			DiffTool:     c.DiffTool,
			DiffToolOpts: c.DiffToolOpts,
			Debug:        c.Debug,
			Output:       c.Output,
		}
	}
}

// PkgDiffer knows how to compare given packages.
type PkgDiffer interface {
	Diff(pkgs ...string) error
}

type defaultPkgDiffer struct {
	// DiffType specifies the type of changes to show
	DiffType Type

	// Difftool refers to diffing commandline tool for showing changes.
	DiffTool string

	// DiffToolOpts refers to the commandline options to for the diffing tool.
	DiffToolOpts string

	// When Debug is true, command will run with verbose logging and will not
	// cleanup the staged packages to assist with debugging.
	Debug bool

	// Output is an io.Writer where command will write the output of the
	// command.
	Output io.Writer
}

func (d *defaultPkgDiffer) Diff(pkgs ...string) error {
	// add merge comments before comparing so that there are no unwanted diffs
	if err := addmergecomment.Process(pkgs...); err != nil {
		return err
	}
	for _, pkg := range pkgs {
		if err := d.prepareForDiff(pkg); err != nil {
			return err
		}
	}
	var args []string
	if d.DiffToolOpts != "" {
		args = strings.Split(d.DiffToolOpts, " ")
		args = append(args, pkgs...)
	} else {
		args = pkgs
	}
	cmd := exec.Command(d.DiffTool, args...)
	cmd.Stdout = d.Output
	cmd.Stderr = d.Output

	if d.Debug {
		fmt.Fprintf(d.Output, "%s\n", strings.Join(cmd.Args, " "))
	}
	err := cmd.Run()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok && exitErr.ExitCode() == 1 {
			// diff tool will exit with return code 1 if there are differences
			// between two dirs. This suppresses those errors.
			err = nil
		} else if ok {
			// An error occurred but was not one of the excluded ones
			// Attempt to display help information to assist with resolving
			fmt.Printf(exitCodeDiffWarning, d.DiffTool, d.DiffType)
		}
	}
	return err
}

// prepareForDiff removes metadata such as .git and Kptfile from a staged package
// to exclude them from diffing.
func (d *defaultPkgDiffer) prepareForDiff(dir string) error {
	excludePaths := []string{".git", kptfilev1.KptFileName}
	for _, path := range excludePaths {
		path = filepath.Join(dir, path)
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

// PkgGetter knows how to fetch a package given a git repo, path and ref.
type PkgGetter interface {
	GetPkg(ctx context.Context, stagingDir, targetDir, repo, path, ref string) (dir string, err error)
}

// defaultPkgGetter uses fetch.Command abstraction to implement PkgGetter.
type defaultPkgGetter struct{}

// GetPkg checks out a repository into a temporary directory for diffing
// and returns the directory containing the checked out package or an error.
// repo is the git repository the package was cloned from.  e.g. https://
// path is the sub directory of the git repository that the package was cloned from
// ref is the git ref the package was cloned from
func (pg defaultPkgGetter) GetPkg(ctx context.Context, stagingDir, targetDir, repo, path, ref string) (string, error) {
	dir, err := stageDirectory(stagingDir, targetDir)
	if err != nil {
		return dir, err
	}

	name := filepath.Base(dir)
	kf := kptfileutil.DefaultKptfile(name)
	kf.Upstream = &kptfilev1.Upstream{
		Type: kptfilev1.GitOrigin,
		Git: &kptfilev1.Git{
			Repo:      repo,
			Directory: path,
			Ref:       ref,
		},
	}
	err = kptfileutil.WriteFile(dir, kf)
	if err != nil {
		return dir, err
	}

	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, dir)
	if err != nil {
		return dir, err
	}

	cmdGet := &fetch.Command{
		Pkg: p,
	}
	err = cmdGet.Run(ctx)
	return dir, err
}

// stageDirectory creates a subdirectory of the provided path for temporary operations
// path is the parent staged directory and should already exist
// subpath is the subdirectory that should be created inside path
func stageDirectory(path, subpath string) (string, error) {
	targetPath := filepath.Join(path, subpath)
	err := os.Mkdir(targetPath, os.ModePerm)
	return targetPath, err
}

// NameStagingDirectory assigns a name that matches the package source information
func NameStagingDirectory(source, ref string) string {
	// Using tags may result in references like /refs/tags/version
	// To avoid creating additional directory's use only the last name after a /
	splitRef := strings.Split(ref, "/")
	reducedRef := splitRef[len(splitRef)-1]

	return fmt.Sprintf("%s-%s",
		source,
		reducedRef)
}
