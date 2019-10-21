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

// Package cmdupdate contains the update command
package cmdupdate

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"kpt.dev/kpt/util/update"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "update LOCAL_PKG_DIR[@VERSION]",
		Short: "Update a local package with changes from a remote source repo",
		Long: `Update a local package with changes from a remote source repo.

Args:

  LOCAL_PKG_DIR:
    Local package to update.  Directory must exist and contain a Kptfile.
    Defaults to the current working directory.

  VERSION:
  	A git tag, branch, ref or commit.  Specified after the local_package with @ -- pkg@version.
    Defaults the local package version that was last fetched.

	Version types:

    * branch: update the local contents to the tip of the remote branch
    * tag: update the local contents to the remote tag
    * commit: update the local contents to the remote commit

Flags:

  --strategy:
    Controls how changes to the local package are handled.  Defaults to fast-forward.

    * resource-merge: perform a structural comparison of the original / updated Resources, and merge
	  the changes into the local package.  See ` + "`kpt help apis merge3` for details on merge." + `
    * fast-forward: fail without updating if the local package was modified since it was fetched.
    * alpha-git-patch: use 'git format-patch' and 'git am' to apply a patch of the
      changes between the source version and destination version.
      **REQUIRES THE LOCAL PACKAGE TO HAVE BEEN COMMITTED TO A LOCAL GIT REPO.**
    * force-delete-replace: THIS WILL WIPE ALL LOCAL CHANGES TO
      THE PACKAGE.  DELETE the local package at local_pkg_dir/ and replace it
      with the remote version.

Env Vars:

  KPT_CACHE_DIR:
    Controls where to cache remote packages when fetching them to update local packages.
    Defaults to ~/.kpt/repos/
`,
		Example: `  # update my-package-dir/
  kpt update my-package-dir/

  # update my-package-dir/ to match the v1.3 branch or tag
  kpt update my-package-dir/@v1.3

  # update applying a git patch
  git add my-package-dir/
  git commit -m "package updates"
  kpt update my-package-dir/@master --strategy alpha-git-patch
`,
		RunE:         r.runE,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		PreRunE:      r.preRunE,
		SuggestFor:   []string{"rebase", "replace"},
	}

	c.Flags().StringVarP(&r.Repo, "repo", "r", "",
		"git repo url for updating contents.  defaults to the repo the package was fetched from.")
	c.Flags().StringVar(&r.strategy, "strategy", string(update.KResourceMerge),
		"update strategy for preserving changes to the local package.")
	c.Flags().BoolVar(&r.DryRun, "dry-run", false,
		"print the git patch rather than merging it.")
	c.Flags().BoolVar(&r.Verbose, "verbose", false,
		"print verbose logging information.")
	r.C = c
	return r
}

// Runner contains the run function.
// TODO, support listing versions
type Runner struct {
	strategy string
	update.Command
	C *cobra.Command
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	r.Command.Strategy = update.StrategyType(r.strategy)
	parts := strings.Split(args[0], "@")
	if len(parts) > 2 {
		return fmt.Errorf("at most 1 version permitted")
	}
	r.Command.Path = parts[0]
	if len(parts) > 1 {
		r.Command.Ref = parts[1]
	}

	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	if len(r.Ref) > 0 {
		fmt.Fprintf(c.OutOrStdout(), "updating package %s to %s\n", r.Command.Path, r.Ref)
	} else {
		fmt.Fprintf(c.OutOrStdout(), "updating package %s\n", r.Command.Path)
	}
	return r.Run()
}
