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

package commands

import (
	"github.com/GoogleContainerTools/kpt/internal/cmddesc"
	"github.com/GoogleContainerTools/kpt/internal/cmddiff"
	"github.com/GoogleContainerTools/kpt/internal/cmdfix"
	"github.com/GoogleContainerTools/kpt/internal/cmdget"
	"github.com/GoogleContainerTools/kpt/internal/cmdinit"
	"github.com/GoogleContainerTools/kpt/internal/cmdsetdependency"
	"github.com/GoogleContainerTools/kpt/internal/cmdsync"
	"github.com/GoogleContainerTools/kpt/internal/cmdupdate"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/spf13/cobra"
)

func GetPkgCommand(name string) *cobra.Command {
	pkg := &cobra.Command{
		Use:     "pkg",
		Short:   pkgdocs.PkgShort,
		Long:    pkgdocs.PkgLong,
		Example: pkgdocs.PkgExamples,
		Aliases: []string{"package"},
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := cmd.Flags().GetBool("help")
			if err != nil {
				return err
			}
			if h {
				return cmd.Help()
			}
			return cmd.Usage()
		},
	}
	pkg.AddCommand(
		cmddesc.NewCommand(name), cmdget.NewCommand(name), cmdinit.NewCommand(name),
		cmdfix.NewCommand(name), cmdsync.NewCommand(name), cmdupdate.NewCommand(name),
		cmddiff.NewCommand(name), cmdsetdependency.NewCommand(name),
	)
	return pkg
}
