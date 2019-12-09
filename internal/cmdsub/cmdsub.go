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

// Package cmdsub contains the sub command.
package cmdsub

import (
	"fmt"
	"path/filepath"
	"strconv"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/commands"
	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/GoogleContainerTools/kpt/internal/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/sub"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

// NewRunner returns a command runner.
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:     "sub PKG_DIR SUBSTITUTION_NAME NEW_VALUE",
		Args:    cobra.RangeArgs(1, 3),
		Short:   docs.SubShort,
		Long:    docs.SubLong,
		Example: docs.SubExamples,
		RunE:    r.runE,
		PreRunE: r.preRunE,
	}
	cmdutil.FixDocs("kpt", parent, c)

	r.Command = c
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

type Runner struct {
	Sub     sub.Sub
	Kptfile kptfile.KptFile
	Command *cobra.Command
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	// available substitutions are in the Kptfile
	var err error
	r.Kptfile, err = kptfileutil.ReadFile(args[0])
	if err != nil {
		return errors.WrapPrefixf(err, "failed reading %s",
			filepath.Join(args[0], kptfile.KptFileName))
	}

	// if args < 3, then we won't do an substitutions and will just print help
	if len(args) != 3 {
		return nil
	}

	// find the substitution matching the one specified by the user
	var found *kptfile.Substitution
	for i := range r.Kptfile.Substitutions {
		s := r.Kptfile.Substitutions[i]
		if s.Name == args[1] {
			// this is the one the user specified
			found = &s
			break
		}
	}
	if found == nil {
		// user specified an invalid substitution -- or one not known to the Kptfile
		return errors.Errorf("no package substitutions matching %s", args[1])
	}

	// init the substitution
	r.Sub.Substitution = *found
	r.Sub.Substitution.StringValue = args[2]

	// validate the input
	if r.Sub.Substitution.Type == kptfile.Int {
		_, err := strconv.Atoi(args[2])
		if err != nil {
			return errors.WrapPrefixf(err, "NEW_VALUE must be an int")
		}
	}
	if r.Sub.Substitution.Type == kptfile.Bool {
		_, err := strconv.ParseBool(args[2])
		if err != nil {
			return errors.WrapPrefixf(err, "NEW_VALUE must be a bool")
		}
	}
	if r.Sub.Substitution.Type == kptfile.Float {
		_, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return errors.WrapPrefixf(err, "NEW_VALUE must be a float")
		}
	}
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	if len(args) != 3 {
		return r.doHelp(c, args)
	}
	return r.doSub(c, args)
}

func (r *Runner) doSub(c *cobra.Command, args []string) error {
	rw := &kio.LocalPackageReadWriter{
		PackagePath: args[0],
	}
	// perform the substitutions in the package
	err := kio.Pipeline{
		Inputs:  []kio.Reader{rw},
		Filters: []kio.Filter{&r.Sub},
		Outputs: []kio.Writer{rw},
	}.Execute()
	if err != nil {
		return err
	}
	fmt.Fprintf(c.OutOrStdout(), "performed %d substitutions\n", r.Sub.Count)
	return nil
}

// doHelp prints help messages for the available substitutions
func (r *Runner) doHelp(c *cobra.Command, args []string) error {
	// use a cobra command to print the help messages

	// create a parent command for the package
	parent := &cobra.Command{
		Use:   args[0],
		Short: "Perform substitutions for Resources in package " + args[0],
		Long:  "Perform substitutions for Resources in package " + args[0],
	}

	// create a command for each available substitution
	for i := range r.Kptfile.Substitutions {
		s := r.Kptfile.Substitutions[i]
		parent.AddCommand(&cobra.Command{
			Use:     s.Name,
			Short:   s.Marker + " (" + string(s.Type) + ") " + s.Short,
			Long:    s.Long,
			Example: s.Example,
			RunE: func(cmd *cobra.Command, args []string) error {
				// don't delete --
				// this function is required to get the command to show up in the help
				return cmd.Help()
			},
		})
	}

	// re-execute the command with the help messaging -- automatically
	// invokes the root command
	c.AddCommand(parent)
	return c.Execute()
}
