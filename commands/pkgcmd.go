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
	"github.com/GoogleContainerTools/kpt/internal/cmddiff"
	"github.com/GoogleContainerTools/kpt/internal/cmdget"
	"github.com/GoogleContainerTools/kpt/internal/cmdinit"
	"github.com/GoogleContainerTools/kpt/internal/cmdupdate"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/cmd/config/configcobra"
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

	tree := configcobra.Tree(name)
	tree.Short = pkgdocs.TreeShort
	tree.Long = pkgdocs.TreeShort + "\n" + pkgdocs.TreeLong
	tree.Example = pkgdocs.TreeExamples

	pkg.AddCommand(
		cmdget.NewCommand(name), cmdinit.NewCommand(name),
		cmdupdate.NewCommand(name), cmddiff.NewCommand(name), tree,
	)
	return pkg
}
