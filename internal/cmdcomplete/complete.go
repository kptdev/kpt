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

// Package cmdcomplete contains the install-completion command
package cmdcomplete

import (
	"os"
	"strings"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/commands"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/update"
	"github.com/posener/complete/v2"
	"github.com/posener/complete/v2/predict"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

// NewRunner returns an install-completion command runner
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:     "install-completion",
		Short:   docs.CompleteShort,
		Long:    docs.CompleteLong,
		Example: docs.CompleteExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
	}
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

// NewCommand returns a new install-completion command
func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

// Runner runs the command
type Runner struct {
	Command *cobra.Command
}

func (Runner) preRunE(cmd *cobra.Command, args []string) error {
	if os.Getenv("COMP_INSTALL") == "" {
		if err := errors.Wrap(os.Setenv("COMP_INSTALL", "1")); err != nil {
			return err
		}
	}
	return nil
}

func (Runner) runE(cmd *cobra.Command, args []string) error {
	// find the root
	for cmd.Parent() != nil {
		cmd = cmd.Parent()
	}
	c := Complete(cmd, nil)
	c.Complete("kpt")
	return nil
}

type VisitFlags func(cmd *cobra.Command, flag *pflag.Flag, cc *complete.Command)

// Complete returns a completion command for a cobra command
func Complete(cmd *cobra.Command, visitFlags VisitFlags) *complete.Command {
	cc := &complete.Command{
		Flags: map[string]complete.Predictor{},
		Sub:   map[string]*complete.Command{},
	}

	// add each command
	for i := range cmd.Commands() {
		c := cmd.Commands()[i]
		name := strings.Split(c.Use, " ")[0]
		cc.Sub[name] = Complete(c, visitFlags)
	}

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if visitFlags != nil {
			// extension support for other commands that embed this one
			visitFlags(cmd, flag, cc)
		}

		if flag.Name == "strategy" {
			cc.Flags[flag.Name] = predict.Options(predict.OptValues(update.Strategies...))
			return
		}
		if flag.Name == "pattern" {
			cc.Flags[flag.Name] = predict.Options(predict.OptValues("%n_%k.yaml"))
			return
		}
		cc.Flags[flag.Name] = predict.Nothing
	})

	return cc
}
