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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
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

	// PkgDiffer specifies package differ
	PkgDiffer PkgDiffer

	// PkgGetter specifies packaging sourcing adapter
	PkgGetter PkgGetter
}

func (c *Command) Run() error {
	c.DefaultValues()

	kptFile, err := kptfileutil.ReadFile(c.Path)
	if err != nil {
		return errors.Errorf("package missing Kptfile at '%s': %v", c.Path, err)
	}

	// Stage current package
	// This prevents prepareForDiff from modifying the local package
	currPkg, err := ioutil.TempDir("", "kpt-local-")
	if err != nil {
		return errors.Errorf("failed to create stage dir for current package: %v", err)
	}
	defer func() {
		if !c.Debug {
			defer os.RemoveAll(currPkg)
		}
	}()

	err = copyutil.CopyDir(c.Path, currPkg)
	if err != nil {
		return errors.Errorf("failed to stage current package: %v", err)
	}
	fmt.Printf("Staging %s at %s\n", c.Path, currPkg)

	// get the upstreamPkg at current version
	upstreamPkg, err := c.PkgGetter.GetPkg(kptFile.Upstream.Git.Repo,
		kptFile.Upstream.Git.Directory,
		kptFile.Upstream.Git.Commit)
	fmt.Printf("Staging %s/%s:%s at %s\n",
		kptFile.Upstream.Git.Repo,
		kptFile.Upstream.Git.Directory,
		"PackageVersion",
		upstreamPkg)
	if err != nil {
		return err
	}
	defer func() {
		if !c.Debug {
			defer os.RemoveAll(upstreamPkg)
		}
	}()

	var upstreamTargetPkg string

	if c.Ref == "" {
		c.Ref, err = gitutil.DefaultRef(kptFile.Upstream.Git.Repo)
		if err != nil {
			return err
		}
	}

	if c.DiffType == DiffTypeRemote ||
		c.DiffType == DiffTypeCombined ||
		c.DiffType == DiffType3Way {
		// get the upstream pkg at the target version
		upstreamTargetPkg, err = c.PkgGetter.GetPkg(kptFile.Upstream.Git.Repo,
			kptFile.Upstream.Git.Directory,
			c.Ref)
		if err != nil {
			return err
		}
		fmt.Printf("Staging %s/%s:%s at %s\n",
			kptFile.Upstream.Git.Repo,
			kptFile.Upstream.Git.Directory,
			c.Ref,
			upstreamTargetPkg)
		defer func() {
			if !c.Debug {
				defer os.RemoveAll(upstreamTargetPkg)
			}
		}()
	}

	if c.Debug {
		fmt.Fprintf(c.Output, "diffing currPkg: %v, upstreamPkg: %v, upstreamTargetPkg: %v \n",
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

func (d *defaultPkgDiffer) Diff(pkgs ...string) error {
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
		if exitErr, ok := err.(*exec.ExitError); ok &&
			exitErr.ExitCode() == 1 {
			// diff tool will exit with return code 1 if there are differences
			// between two dirs. This suppresses those errors.
			err = nil
		}
	}
	return err
}

// prepareForDiff removes metadata such as .git and Kptfile from a staged package
// to exclude them from diffing.
func (d *defaultPkgDiffer) prepareForDiff(dir string) error {
	excludePaths := []string{".git", kptfile.KptFileName}
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
	GetPkg(repo, path, ref string) (dir string, err error)
}

// defaultPkgGetter uses get.Command abstraction to implement PkgGetter.
type defaultPkgGetter struct{}

// GetPkg checks out a repository into a temporary directory for diffing
// and returns the directory containing the checked out package or an error.
// repo is the git repository the package was cloned from.  e.g. https://
// path is the sub directory of the git repository that the package was cloned from
// ref is the git ref the package was cloned from
func (pg defaultPkgGetter) GetPkg(repo, path, ref string) (string, error) {
	repoSrc := strings.Split(repo, "/") // For github repo's this will be the project name
	pkgSrc := strings.Split(path, "/")  // This will be the directory the package is contained in
	tmpPath := fmt.Sprintf("kpt-upstream-%s-%s-",
		repoSrc[len(repoSrc)-1],
		pkgSrc[len(pkgSrc)-1])
	dir, err := ioutil.TempDir("", tmpPath)
	if err != nil {
		return dir, err
	}
	cmdGet := &get.Command{
		Git:         kptfile.Git{Repo: repo, Directory: path, Ref: ref},
		Destination: dir,
		Clean:       true,
	}
	err = cmdGet.Run()
	return dir, err
}
