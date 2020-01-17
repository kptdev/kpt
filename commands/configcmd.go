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
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/cfgdocs"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/cmd/config/configcobra"
)

func GetConfigCommand(name string) *cobra.Command {
	cfgCmd := &cobra.Command{
		Use:     "cfg",
		Short:   cfgdocs.READMEShort,
		Long:    cfgdocs.READMEShort + "\n" + cfgdocs.READMELong,
		Example: cfgdocs.READMEExamples,
		Aliases: []string{"config"},
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

	cat := configcobra.Cat(name)
	cat.Short = cfgdocs.CatShort
	cat.Long = cfgdocs.CatShort + "\n" + cfgdocs.CatLong
	cat.Example = cfgdocs.CatExamples

	count := configcobra.Count(name)
	count.Short = cfgdocs.CountShort
	count.Long = cfgdocs.CountShort + "\n" + cfgdocs.CountLong
	count.Example = cfgdocs.CountExamples

	createSetter := configcobra.CreateSetter(name)
	createSetter.Short = cfgdocs.CreateSetterShort
	createSetter.Long = cfgdocs.CreateSetterShort + "\n" + cfgdocs.CreateSetterLong
	createSetter.Example = cfgdocs.CreateSetterExamples

	fmt := configcobra.Fmt(name)
	fmt.Short = cfgdocs.FmtShort
	fmt.Long = cfgdocs.FmtShort + "\n" + cfgdocs.FmtLong
	fmt.Example = cfgdocs.FmtExamples

	grep := configcobra.Grep(name)
	grep.Short = cfgdocs.GrepShort
	grep.Long = cfgdocs.GrepShort + "\n" + cfgdocs.GrepLong
	grep.Example = cfgdocs.GrepExamples

	listSetters := configcobra.ListSetters(name)
	listSetters.Short = cfgdocs.ListSettersShort
	listSetters.Long = cfgdocs.ListSettersShort + "\n" + cfgdocs.ListSettersLong
	listSetters.Example = cfgdocs.ListSettersExamples

	merge := configcobra.Merge(name)
	merge.Short = cfgdocs.MergeShort
	merge.Long = cfgdocs.MergeShort + "\n" + cfgdocs.MergeLong
	merge.Example = cfgdocs.MergeExamples

	merge3 := configcobra.Merge3(name)
	merge3.Short = cfgdocs.Merge3Short
	merge3.Long = cfgdocs.Merge3Short + "\n" + cfgdocs.Merge3Long
	merge3.Example = cfgdocs.Merge3Examples

	set := configcobra.Set(name)
	set.Short = cfgdocs.SetShort
	set.Long = cfgdocs.SetShort + "\n" + cfgdocs.SetLong
	set.Example = cfgdocs.SetExamples

	tree := configcobra.Tree(name)
	tree.Short = cfgdocs.TreeShort
	tree.Long = cfgdocs.TreeShort + "\n" + cfgdocs.TreeLong
	tree.Example = cfgdocs.TreeExamples

	cfgCmd.AddCommand(cat, count, createSetter, fmt, grep, listSetters, merge, merge3, set, tree)
	return cfgCmd
}
