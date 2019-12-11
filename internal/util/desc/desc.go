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

// Package desc contains libraries for printing package descriptions.
package desc

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/GoogleContainerTools/kpt/internal/kptfile/kptfileutil"
	"github.com/olekukonko/tablewriter"
)

// Command prints information about the given packages.
type Command struct {
	// StdOut is the StdOut value
	StdOut io.Writer

	// PkgPaths refers to the pkg directories to be described.
	PkgPaths []string

	PrintBasePath bool
}

// Run prints information about given packages in a tabular format.
// A directory containing KptFile is considered to be a valid package.
// Invalid packages are ignored.
func (c Command) Run() error {
	var pkgs []pkgInfo
	for _, p := range c.PkgPaths {
		err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Name() != kptfile.KptFileName {
				return nil
			}
			kptFile, err := kptfileutil.ReadFile(filepath.Dir(path))
			if err != nil {
				return nil
			}
			path = filepath.Clean(path)
			fmt.Println(path)
			pkgs = append(pkgs, pkgInfo{localDir: path, KptFile: kptFile})
			return nil
		})
		if err != nil {
			return err
		}
	}

	c.printPkgs(c.GetStdOut(), pkgs)
	return nil
}

// GetStdOut returns the io.Writer that will be used as describe stdout.
func (c Command) GetStdOut() io.Writer {
	if c.StdOut == nil {
		return os.Stdout
	}
	return c.StdOut
}

func (c Command) printPkgs(w io.Writer, pkgs []pkgInfo) {
	table := tablewriter.NewWriter(w)
	table.SetRowLine(false)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetColumnSeparator(" ")
	table.SetCenterSeparator(" ")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{
		"Package Name", "Dir", "Remote",
		"Remote Path", "Remote Ref", "Remote Commit"})
	for _, pkg := range pkgs {
		p := filepath.Dir(pkg.localDir)
		if c.PrintBasePath {
			p = filepath.Base(p)
		}
		table.Append([]string{
			pkg.Name,
			p,
			pkg.Upstream.Git.Repo,
			pkg.Upstream.Git.Directory,
			pkg.Upstream.Git.Ref,
			shortSHA(pkg.Upstream.Git.Commit),
		})
	}
	table.Render()
}

// shortSHA returns short form (first 7 letters) of the commit SHA.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// pkgInfo wraps KptFile with local directory path info.
type pkgInfo struct {
	localDir string
	kptfile.KptFile
}
