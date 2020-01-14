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
	configdocs "github.com/GoogleContainerTools/kpt/internal/docs/generated/config"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/cmd/config/configcobra"
)

func GetConfigCommand(name string) *cobra.Command {
	cfgCmd := &cobra.Command{
		Use:     "config",
		Short:   configdocs.READMEShort,
		Long:    configdocs.READMEShort + "\n" + configdocs.READMELong,
		Example: configdocs.READMEExamples,
	}

	cat := configcobra.Cat(name)
	cat.Short = configdocs.CatShort
	cat.Long = configdocs.CatShort + "\n" + configdocs.CatLong
	cat.Example = configdocs.CatExamples

	count := configcobra.Count(name)
	count.Short = configdocs.CountShort
	count.Long = configdocs.CountShort + "\n" + configdocs.CountLong
	count.Example = configdocs.CountExamples

	createSetter := configcobra.CreateSetter(name)
	createSetter.Short = configdocs.CreateSetterShort
	createSetter.Long = configdocs.CreateSetterShort + "\n" + configdocs.CreateSetterLong
	createSetter.Example = configdocs.CreateSetterExamples

	fmt := configcobra.Fmt(name)
	fmt.Short = configdocs.FmtShort
	fmt.Long = configdocs.FmtShort + "\n" + configdocs.FmtLong
	fmt.Example = configdocs.FmtExamples

	grep := configcobra.Grep(name)
	grep.Short = configdocs.GrepShort
	grep.Long = configdocs.GrepShort + "\n" + configdocs.GrepLong
	grep.Example = configdocs.GrepExamples

	listSetters := configcobra.ListSetters(name)
	listSetters.Short = configdocs.ListSettersShort
	listSetters.Long = configdocs.ListSettersShort + "\n" + configdocs.ListSettersLong
	listSetters.Example = configdocs.ListSettersExamples

	merge := configcobra.Merge(name)
	merge.Short = configdocs.MergeShort
	merge.Long = configdocs.MergeShort + "\n" + configdocs.MergeLong
	merge.Example = configdocs.MergeExamples

	merge3 := configcobra.Merge3(name)
	merge3.Short = configdocs.Merge3Short
	merge3.Long = configdocs.Merge3Short + "\n" + configdocs.Merge3Long
	merge3.Example = configdocs.Merge3Examples

	set := configcobra.Set(name)
	set.Short = configdocs.SetShort
	set.Long = configdocs.SetShort + "\n" + configdocs.SetLong
	set.Example = configdocs.SetExamples

	tree := configcobra.Tree(name)
	tree.Short = configdocs.TreeShort
	tree.Long = configdocs.TreeShort + "\n" + configdocs.TreeLong
	tree.Example = configdocs.TreeExamples

	cfgCmd.AddCommand(cat, count, createSetter, fmt, grep, listSetters, merge, merge3, set, tree)
	return cfgCmd
}
