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

// Package cmdsync contains the sync command
package cmdsync

import (
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/sync"
	"github.com/spf13/cobra"
)

// NewRunner returns a command runner.
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:     "sync LOCAL_PKG_DIR",
		Short:   SyncShort,
		Long:    SyncLong,
		Example: SyncExamples,
		RunE:    r.runE,
		Args:    cobra.ExactArgs(1),
		PreRunE: r.preRunE,
	}

	c.Flags().BoolVar(&r.Sync.Verbose, "verbose", false,
		"print verbose logging information.")
	c.Flags().BoolVar(&r.Sync.DryRun, "dry-run", false,
		"print sync actions without performing them.")
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c

	c.AddCommand(NewSetCommand(parent))
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

type Runner struct {
	Sync    sync.Command
	Command *cobra.Command
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	r.Sync.Dir = args[0]
	r.Sync.StdOut = c.OutOrStdout()
	r.Sync.StdErr = c.ErrOrStderr()
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	return r.Sync.Run()
}

// This content is no longer available on the Hugo site, so we just put
// it here for now.

var SyncShort = `Fetch and update packages declaratively`
var SyncLong = `
Sync uses a manifest to manage a collection of dependencies.

The manifest declares *all* direct dependencies of a package in a Kptfile.
When ` + "`" + `sync` + "`" + ` is run, it will ensure each dependency has been fetched at the
specified ref.

This is an alternative to managing package dependencies individually using
the ` + "`" + `get` + "`" + ` and ` + "`" + `update` + "`" + ` commands.

| Command  | Description                             |
|----------|-----------------------------------------|
| [set]    | add a sync dependency to a Kptfile      |

#### Run Sync

    kpt pkg sync LOCAL_PKG_DIR [flags]

  LOCAL_PKG_DIR:

    Local package with dependencies to sync.  Directory must exist and contain a Kptfile.

#### Env Vars

  KPT_CACHE_DIR:

    Controls where to cache remote packages during updates.
    Defaults to ~/.kpt/repos/

#### Dependencies

For each dependency in the Kptfile, ` + "`" + `sync` + "`" + ` will ensure that it exists locally with the
matching repo and ref.

Dependencies are specified in the ` + "`" + `Kptfile` + "`" + ` ` + "`" + `dependencies` + "`" + ` field and can be added or updated
with ` + "`" + `kpt pkg sync set` + "`" + `.  e.g.

    kpt pkg sync set https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set \
        hello-world
Note that the [set] command must be run from within the local package directory and the
last argument specifies the local destination directory for the dependency.

Or edit the Kptfile directly:

    apiVersion: kpt.dev/v1alpha1
    kind: Kptfile
    dependencies:
    - name: hello-world
      git:
        repo: "https://github.com/GoogleContainerTools/kpt.git"
        directory: "/package-examples/helloworld-set"
        ref: "master"

Dependencies have following schema:

    name: <local path (relative to the Kptfile) to fetch the dependency to>
    git:
      repo: <git repository>
      directory: <sub-directory under the git repository>
      ref: <git reference -- e.g. tag, branch, commit, etc>
    updateStrategy: <strategy to use when updating the dependency -- see kpt help update for more details>
    ensureNotExists: <remove the dependency, mutually exclusive with git>

Dependencies maybe be updated by updating their ` + "`" + `git.ref` + "`" + ` field and running ` + "`" + `kpt pkg sync` + "`" + `
against the directory.
`
var SyncExamples = `
  Example Kptfile to sync:

    # file: my-package-dir/Kptfile

    apiVersion: kpt.dev/v1alpha1
    kind: Kptfile
    # list of dependencies to sync
    dependencies:
    - name: local/destination/dir
      git:
        # repo is the git repository
        repo: "https://github.com/pwittrock/examples"
        # directory is the git subdirectory
        directory: "staging/cockroachdb"
        # ref is the ref to fetch
        ref: "v1.0.0"
    - name: local/destination/dir1
      git:
        repo: "https://github.com/pwittrock/examples"
        directory: "staging/javaee"
        ref: "v1.0.0"
      # set the strategy for applying package updates
      updateStrategy: "resource-merge"
    - name: app2
      path: local/destination/dir2
      # declaratively delete this dependency
      ensureNotExists: true

  Example invocation:

    # print the dependencies that would be modified
    kpt pkg sync my-package-dir/ --dry-run

    # sync the dependencies
    kpt pkg sync my-package-dir/

[tutorial-script]: ../gifs/pkg-sync.sh
[sync-set]: sync-set.md
`
