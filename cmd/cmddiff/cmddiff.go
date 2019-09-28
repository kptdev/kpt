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

	"github.com/spf13/cobra"
	"kpt.dev/util/argutil"
	"kpt.dev/util/diff"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "diff LOCAL_PKG_DIR[@VERSION]",
		Short: "Show changes between local and upstream source package.",
		Long: `Show changes between local and upstream source package.

Diff commands lets you answer the following questions:
  - What have I changed in my package relative to the upstream source package
  - What has changed in the upstream source package between the original source version and target version

You can specify a diffing tool with options to show the changes. By default, it
uses 'diff' commandline tool.

Args:

  LOCAL_PKG_DIR:
    Local package to compare. Command will fail if the directory doesn't exist, or does not
    contain a Kptfile.  Defaults to the current working directory.

  VERSION:
    A git tag, branch, ref or commit. Specified after the local_package with @ -- pkg_dir@version.
    Defaults to the local package version that was last fetched.

Envs:

  KPT_EXTERNAL_DIFF:
   Commandline diffing tool (diff by default) that will be used to show changes. For ex.
   # Use meld to show changes
   KPT_EXTERNAL_DIFF=meld kpt diff

  KPT_EXTERNAL_DIFF_OPTS:
   Commandline options to use for the diffing tool. For ex.
   # Using "-a" diff option
   KPT_EXTERNAL_DIFF_OPTS="-a" kpt diff --diff-tool meld

Flags:
  diff-type:
    The type of changes to view (local by default). Following types are supported:
	 local: shows changes in local package relative to upstream source package at original version
	 remote: shows changes in upstream source package at target version relative to original version
	 combined: shows changes in local package relative to upstream source package at target version
	 3way: shows changes in local package and source package at target version relative to original version side by side

  diff-tool:
    Commandline tool (diff by default) for showing the changes.
	# Show changes using 'meld' commandline tool
	kpt diff @master --diff-tool meld

	Note that it overrides the KPT_EXTERNAL_DIFF environment variable.

  diff-opts:
    Commandline options to use with the diffing tool.
	# Show changes using "diff" with recurive options
	kpt diff @master --diff-tool meld --diff-opts "-r"

	Note that it overrides the KPT_EXTERNAL_DIFF_OPTS environment variable.

`,
		Example: `  # Show changes in current package relative to upstream source package
    kpt diff

    # Show changes in current package relative to upstream source package using meld tool with auto compare option.
    kpt diff --diff-tool meld --diff-tool-opts "-a"

    # Show changes in upstream source package between current version and target version
    kpt diff @v4.0.0 --diff-type remote

    # Show changes in current package relative to target version
    kpt diff @v4.0.0 --diff-type combined

    # Show 3way changes between the local package, upstream package at original version and upstream package at target version using meld
    kpt diff @v4.0.0 --diff-type 3way --diff-tool meld --diff-tool-opts "-a"
`,
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
	return r
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
