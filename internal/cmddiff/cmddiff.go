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

// Package cmddiff contains the diff command
package cmddiff

import (
	"os"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/diff"
	"github.com/spf13/cobra"
)

// NewRunner returns a command runner.
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:          "diff LOCAL_PKG_DIR[@VERSION]",
		Short:        docs.DiffShort,
		Long:         docs.DiffLong,
		Example:      docs.DiffExamples,
		PreRunE:      r.preRunE,
		RunE:         r.runE,
		SilenceUsage: true,
	}
	diffTool := "diff"
	if tool := os.Getenv("KPT_EXTERNAL_DIFF"); tool != "" {
		diffTool = tool
	}
	diffToolOpts := os.Getenv("KPT_EXTERNAL_DIFF_OPTS")
	c.Flags().StringVar(&r.diffType, "diff-type", string(diff.DiffTypeLocal),
		"diff type you want to perform e.g. "+diff.SupportedDiffTypesLabel())
	c.Flags().StringVar(&r.DiffTool, "diff-tool", diffTool,
		"diff tool to use to show the changes")
	c.Flags().StringVar(&r.DiffToolOpts, "diff-tool-opts", diffToolOpts,
		"diff tool commandline options to use to show the changes")
	c.Flags().BoolVar(&r.Debug, "debug", false,
		"when true, prints additional debug information and do not delete staged pkg dirs")
	r.C = c
	r.Output = c.OutOrStdout()
	cmdutil.FixDocs("kpt", parent, c)
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).C
}

// Runner contains the run function
type Runner struct {
	diff.Command
	C        *cobra.Command
	diffType string
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	dirVer := ""
	if len(args) > 0 {
		dirVer = args[0]
	}
	dir, version, err := argutil.ParseDirVersionWithDefaults(dirVer)
	if err != nil {
		return err
	}
	r.Path = dir
	r.Ref = version
	r.DiffType = diff.DiffType(r.diffType)
	return r.Validate()
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	return r.Run()
}
