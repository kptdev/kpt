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

package runner

import (
	"fmt"
	"os"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/go-errors/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/pathutil"
)

// CmdRunner interface holds ExecuteCmd definition which executes respective command's
// implementation on single package
type CmdRunner interface {
	ExecuteCmd(pkgPath string) error
}

// ExecuteCmdOnPkgs struct holds the parameters necessary to
// execute the filter command on packages in rootPkgPath
type ExecuteCmdOnPkgs struct {
	RootPkgPath        string
	RecurseSubPackages bool
	NeedKptFile        bool
	CmdRunner          CmdRunner
}

// ExecuteCmdOnPkgs takes the function definition for a command to be executed on single package, applies that definition
// recursively on all the subpackages present in rootPkgPath if recurseSubPackages is true, else applies the command on rootPkgPath only
func (e ExecuteCmdOnPkgs) Execute() error {
	pkgsPaths, err := pathutil.DirsWithFile(e.RootPkgPath, kptfile.KptFileName, e.RecurseSubPackages)
	if err != nil {
		return err
	}

	if len(pkgsPaths) == 0 {
		// at this point, there are no openAPI files in the rootPkgPath
		if e.NeedKptFile {
			// few executions need openAPI file to be present(ex: setters commands), if true throw an error
			return errors.Errorf("unable to find %q in package %q", kptfile.KptFileName, e.RootPkgPath)
		}

		// add root path for commands which doesn't need openAPI(ex: annotate, fmt)
		pkgsPaths = []string{e.RootPkgPath}
	}

	// for commands which doesn't need openAPI file, make sure that the root package is
	// included all the times
	if !e.NeedKptFile && !containsString(pkgsPaths, e.RootPkgPath) {
		pkgsPaths = append([]string{e.RootPkgPath}, pkgsPaths...)
	}

	for _, pkgPath := range pkgsPaths {
		err := e.CmdRunner.ExecuteCmd(pkgPath)
		if err != nil {
			return err
		}
	}
	return nil
}

// containsString returns true if slice contains s
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func HandleError(c *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	if StackOnError {
		if err, ok := err.(*errors.Error); ok {
			fmt.Fprintf(os.Stderr, "%s", err.Stack())
		}
	}

	if ExitOnError {
		fmt.Fprintf(c.ErrOrStderr(), "Error: %v\n", err)
		os.Exit(1)
	}
	return err
}

// ExitOnError if true, will cause commands to call os.Exit instead of returning an error.
// Used for skipping printing usage on failure.
var ExitOnError bool

// StackOnError if true, will print a stack trace on failure.
var StackOnError bool
