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

// Package diff contains libraries for diffing packages.
package diff

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/addmergecomment"
	"github.com/GoogleContainerTools/kpt/internal/util/fetch"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/content"
	"github.com/GoogleContainerTools/kpt/pkg/content/open"
	"github.com/GoogleContainerTools/kpt/pkg/content/provider/dir"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/pkg/location"
	"github.com/GoogleContainerTools/kpt/pkg/location/mutate"
	"sigs.k8s.io/kustomize/kyaml/errors"

	"github.com/GoogleContainerTools/kpt/internal/migration/os"
	"github.com/GoogleContainerTools/kpt/internal/migration/path/filepath"
)

// DiffType represents type of comparison to be performed.
type DiffType string

const (
	// DiffTypeLocal shows the changes in local pkg relative to upstream source pkg at original version
	DiffTypeLocal DiffType = "local"
	// DiffTypeRemote shows changes in the upstream source pkg between original and target version
	DiffTypeRemote DiffType = "remote"
	// DiffTypeCombined shows changes in local pkg relative to upstream source pkg at target version
	DiffTypeCombined DiffType = "combined"
	// 3way shows changes in local and remote changes side-by-side
	DiffType3Way DiffType = "3way"
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
func (dt DiffType) String() string {
	return string(dt)
}

var SupportedDiffTypes = []DiffType{DiffTypeLocal, DiffTypeRemote, DiffTypeCombined, DiffType3Way}

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
	DiffType DiffType

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

	// PkgGetter specifies package getter
	PkgGetter PkgGetter

	// PkgDiffer specifies package differ
	PkgDiffer PkgDiffer
}

func (c *Command) Run(ctx context.Context) error {
	c.DefaultValues()

	localRef := location.Dir{Directory: c.Path}
	local, err := open.FileSystem(localRef)
	if err != nil {
		return errors.Errorf("unable to open %q: %v", c.Path, err)
	}
	defer local.Close()
	localPath := local.FileSystemPath

	// Read Kptfile and get upstream ref
	kptFile, err := pkg.ReadKptfile(localPath)
	if err != nil {
		return errors.Errorf("package missing Kptfile at '%s': %v", c.Path, err)
	}
	upstream, err := kptfileutil.NewReferenceFromUpstream(kptFile)
	if err != nil {
		return errors.Errorf("upstream required: %v", err)
	}
	upstreamIdentifier, ok := location.Identifier(upstream)
	if !ok {
		return errors.Errorf("upstream ref required: %w", errors.Errorf("identified not supported by %v", upstream))
	}

	// Create a staging directory to store all compared packages
	temp, err := dir.Temp("kpt-")
	if err != nil {
		return errors.Errorf("failed to create stage dir: %v", err)
	}
	defer temp.Close()
	stagingDirectory, err := content.FileSystem(temp)
	if err != nil {
		return errors.Errorf("failed to create stage dir: %v", err)
	}

	// Stage current package
	// This prevents prepareForDiff from modifying the local package
	localPkgName := NameStagingDirectory(LocalPackageSource, upstreamIdentifier)
	currPkg, err := stageDirectory(stagingDirectory, localPkgName)
	if err != nil {
		return errors.Errorf("failed to create stage dir for current package: %v", err)
	}
	err = pkgutil.CopyPackage(localPath, currPkg, true, pkg.Local)
	if err != nil {
		return errors.Errorf("failed to stage current package: %v", err)
	}
	// get the upstreamPkg at current version
	upstreamPkgName := NameStagingDirectory(RemotePackageSource, upstreamIdentifier)
	upstreamPkg, err := c.PkgGetter.GetPkg(ctx, stagingDirectory, upstreamPkgName, upstream)
	if err != nil {
		return err
	}

	var upstreamTargetPkg types.FileSystemPath

	if c.Ref == "" {
		upstreamLock, err := kptfileutil.NewReferenceLockFromUpstreamLock(kptFile)
		if err != nil {
			return err
		}

		defaultIdentifier, err := location.DefaultIdentifier(upstreamLock, location.WithContext(ctx))
		if err != nil {
			return err
		}

		c.Ref = defaultIdentifier
	}

	var upstreamTarget location.Reference
	if c.DiffType == DiffTypeRemote ||
		c.DiffType == DiffTypeCombined ||
		c.DiffType == DiffType3Way {

		// get the upstream pkg at the target version
		upstreamTarget, err = mutate.Identifier(upstream, c.Ref)
		if err != nil {
			return err
		}
		upstreamTargetPkgName := NameStagingDirectory(TargetRemotePackageSource, c.Ref)
		upstreamTargetPkg, err = c.PkgGetter.GetPkg(ctx, stagingDirectory, upstreamTargetPkgName, upstreamTarget)
		if err != nil {
			return err
		}
	}

	if c.Debug {
		fmt.Fprintf(c.Output, "diffing currPkg:   %v\nupstreamPkg:       %v\nupstreamTargetPkg: %v\n",
			currPkg, upstreamPkg, upstreamTargetPkg)
	}

	switch c.DiffType {
	case DiffTypeLocal:
		return c.PkgDiffer.Diff(currPkg, upstreamPkg)
	case DiffTypeRemote:
		return c.PkgDiffer.Diff(upstreamPkg, upstreamTargetPkg)
	case DiffTypeCombined:
		return c.PkgDiffer.Diff(currPkg, upstreamTargetPkg)
	case DiffType3Way:
		return c.PkgDiffer.Diff(currPkg, upstreamPkg, upstreamTargetPkg)
	default:
		return errors.Errorf("unsupported diff type '%s'", c.DiffType)
	}
}

func (c *Command) Validate() error {
	switch c.DiffType {
	case DiffTypeLocal, DiffTypeCombined, DiffTypeRemote, DiffType3Way:
	default:
		return errors.Errorf("invalid diff-type '%s'. Supported diff-types are: %s",
			c.DiffType, SupportedDiffTypesLabel())
	}

	path, err := exec.LookPath(c.DiffTool)
	if err != nil {
		return errors.Errorf("diff-tool '%s' not found in the PATH.", c.DiffTool)
	}
	c.DiffTool = path
	return nil
}

// DefaultValues sets up the default values for the command.
func (c *Command) DefaultValues() {
	if c.Output == nil {
		c.Output = os.Stdout()
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
	Diff(pkgs ...types.FileSystemPath) error
}

type defaultPkgDiffer struct {
	// DiffType specifies the type of changes to show
	DiffType DiffType

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

func (d *defaultPkgDiffer) Diff(pkgs ...types.FileSystemPath) error {
	// add merge comments before comparing so that there are no unwanted diffs
	if err := addmergecomment.Process(pkgs...); err != nil {
		return err
	}
	var realPaths []string
	for _, pkg := range pkgs {
		if err := d.prepareForDiff(pkg); err != nil {
			return err
		}

		realPaths = append(realPaths, pkg.Path)
	}
	var args []string
	if d.DiffToolOpts != "" {
		args = strings.Split(d.DiffToolOpts, " ")
		args = append(args, realPaths...)
	} else {
		args = realPaths
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
func (d *defaultPkgDiffer) prepareForDiff(dir types.FileSystemPath) error {
	excludePaths := []string{".git", kptfilev1.KptFileName}
	for _, path := range excludePaths {
		rmpath := filepath.Join(dir, path)
		if err := os.RemoveAll(rmpath); err != nil {
			return err
		}
	}
	return nil
}

// PkgGetter knows how to fetch a package given a git repo, path and ref.
type PkgGetter interface {
	GetPkg(ctx context.Context, dst types.FileSystemPath, name string, src location.Reference) (types.FileSystemPath, error)
}

// defaultPkgGetter uses fetch.Command abstraction to implement PkgGetter.
type defaultPkgGetter struct{}

// GetPkg checks out a repository into a temporary directory for diffing
// and returns the directory containing the checked out package or an error.
// repo is the git repository the package was cloned from.  e.g. https://
// path is the sub directory of the git repository that the package was cloned from
// ref is the git ref the package was cloned from
func (pg defaultPkgGetter) GetPkg(ctx context.Context, dst types.FileSystemPath, name string, src location.Reference) (types.FileSystemPath, error) {

	dst, err := stageDirectory(dst, name)
	if err != nil {
		return dst, err
	}

	kf := kptfileutil.DefaultKptfile(name)

	upstream, err := kptfileutil.NewUpstreamFromReference(src, "")
	if err != nil {
		return dst, err
	}

	kf.Upstream = upstream
	err = kptfileutil.WriteFileFS(dst, kf)
	if err != nil {
		return dst, err
	}

	p, err := pkg.New(dst.FileSystem, dst.Path)
	if err != nil {
		return dst, err
	}

	cmdGet := &fetch.Command{
		Pkg: p,
	}

	err = cmdGet.Run(ctx)
	return dst, err
}

// stageDirectory creates a subdirectory of the provided path for temporary operations
// path is the parent staged directory and should already exist
// subpath is the subdirectory that should be created inside path
func stageDirectory(path types.FileSystemPath, subpath string) (types.FileSystemPath, error) {
	targetPath := types.Join(path, subpath)
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
