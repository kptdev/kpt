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

// Package cmdreconcile contains the reconcile command
package cmdreconcile

import (
	"kpt.dev/internal/reconcile"

	"github.com/spf13/cobra"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	r.C = &cobra.Command{
		Use:   "reconcile DIR/",
		Short: "Reconcile runs transformers against the package Resources",
		Long: `Reconcile runs transformers against the package Resources.

  DIR:
    Path to local package directory.

See 'kpt help apis transformers' for more information.
`,
		Example: `# reconcile package transformers
kpt reconcile my-package/
`,
		RunE:         r.runE,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
	}
	return r
}

// Runner contains the run function
type Runner struct {
	IncludeSubpackages bool
	C                  *cobra.Command
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	return reconcile.Cmd{PkgPath: args[0]}.Execute()
}
