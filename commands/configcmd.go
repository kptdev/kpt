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
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/cmdsearch"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/cfgdocs"
	"github.com/GoogleContainerTools/kpt/internal/util/setters"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/cmd/config/configcobra"
	"sigs.k8s.io/kustomize/kyaml/fieldmeta"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyamlsetters "sigs.k8s.io/kustomize/kyaml/setters"
)

const ShortHandRef = "$kpt-set"

func GetConfigCommand(name string) *cobra.Command {
	cfgCmd := &cobra.Command{
		Use:     "cfg",
		Short:   cfgdocs.CfgShort,
		Long:    cfgdocs.CfgShort + "\n" + cfgdocs.CfgLong,
		Example: cfgdocs.CfgExamples,
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

	an := configcobra.Annotate(name)
	an.Short = cfgdocs.AnnotateShort
	an.Long = cfgdocs.AnnotateShort + "\n" + cfgdocs.AnnotateLong
	an.Example = cfgdocs.AnnotateExamples

	cat := configcobra.Cat(name)
	cat.Short = cfgdocs.CatShort
	cat.Long = cfgdocs.CatShort + "\n" + cfgdocs.CatLong
	cat.Example = cfgdocs.CatExamples

	count := configcobra.Count(name)
	count.Short = cfgdocs.CountShort
	count.Long = cfgdocs.CountShort + "\n" + cfgdocs.CountLong
	count.Example = cfgdocs.CountExamples

	createSetter := CreateSetterCommand(name)
	createSetter.Short = cfgdocs.CreateSetterShort
	createSetter.Long = cfgdocs.CreateSetterShort + "\n" + cfgdocs.CreateSetterLong
	createSetter.Example = cfgdocs.CreateSetterExamples

	deleteSetter := DeleteSetterCommand(name)
	deleteSetter.Short = cfgdocs.DeleteSetterShort
	deleteSetter.Long = cfgdocs.DeleteSetterShort + "\n" + cfgdocs.DeleteSetterLong
	deleteSetter.Example = cfgdocs.DeleteSetterExamples

	deleteSubstitution := DeleteSubstitutionCommand(name)
	deleteSubstitution.Short = cfgdocs.DeleteSubstShort
	deleteSubstitution.Long = cfgdocs.DeleteSubstShort + "\n" + cfgdocs.DeleteSubstLong
	deleteSubstitution.Example = cfgdocs.DeleteSubstExamples

	createSubstitution := CreateSubstCommand(name)
	createSubstitution.Short = cfgdocs.CreateSubstShort
	createSubstitution.Long = cfgdocs.CreateSubstShort + "\n" + cfgdocs.CreateSubstLong
	createSubstitution.Example = cfgdocs.CreateSubstExamples

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

	set := SetCommand(name)

	tree := configcobra.Tree(name)
	tree.Short = cfgdocs.TreeShort
	tree.Long = cfgdocs.TreeShort + "\n" + cfgdocs.TreeLong
	tree.Example = cfgdocs.TreeExamples

	search := cmdsearch.SearchCommand(name)
	search.Short = cfgdocs.SearchShort
	search.Long = cfgdocs.SearchShort + "\n" + cfgdocs.SearchLong
	search.Example = cfgdocs.SearchExamples

	cfgCmd.AddCommand(an, cat, count, createSetter, deleteSetter, deleteSubstitution, createSubstitution, fmt,
		grep, listSetters, set, tree, search)

	return cfgCmd
}

func CreateSetterCommand(parent string) *cobra.Command {
	fieldmeta.SetShortHandRef(ShortHandRef)
	return configcobra.CreateSetter(parent)
}

func DeleteSetterCommand(parent string) *cobra.Command {
	fieldmeta.SetShortHandRef(ShortHandRef)
	return configcobra.DeleteSetter(parent)
}

func DeleteSubstitutionCommand(parent string) *cobra.Command {
	fieldmeta.SetShortHandRef(ShortHandRef)
	return configcobra.DeleteSubstitution(parent)
}

func CreateSubstCommand(parent string) *cobra.Command {
	fieldmeta.SetShortHandRef(ShortHandRef)
	return configcobra.CreateSubstitution(parent)
}

// SetCommand wraps the kustomize set command in order to automatically update
// a project number if a project id is set.
func SetCommand(parent string) *cobra.Command {
	fieldmeta.SetShortHandRef(ShortHandRef)
	kustomizeCmd := configcobra.Set(parent)
	setCmd := *kustomizeCmd
	kustomizeCmd.Short = cfgdocs.SetShort
	kustomizeCmd.Long = cfgdocs.SetShort + "\n" + cfgdocs.SetLong
	kustomizeCmd.Example = cfgdocs.SetExamples
	kustomizeCmd.SilenceUsage = true
	kustomizeCmd.SilenceErrors = true
	setCmd.RunE = func(c *cobra.Command, args []string) error {
		warnIfSetterV1(args[0])
		kustomizeCmd.SetArgs(args)
		if err := kustomizeCmd.Execute(); err != nil {
			return err
		}

		if len(args) != 3 || args[1] != setters.GcloudProject {
			return nil
		}

		if setters.DefExists(args[0], setters.GcloudProjectNumber) {
			projectNumber, err := setters.GetProjectNumberFromProjectID(args[2])
			if err != nil {
				return nil
			}
			kustomizeCmd.SetArgs([]string{args[0], setters.GcloudProjectNumber, projectNumber})
			return kustomizeCmd.Execute()
		}
		return nil
	}
	return &setCmd
}

// warnIfSetterV1 checks if the package is using V1 kyaml setters and prints
// warning message to upgrade them using kpt pkg fix command
func warnIfSetterV1(pkgPath string) {
	l := kyamlsetters.LookupSetters{}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.LocalPackageReader{PackagePath: pkgPath}},
		Filters: []kio.Filter{&l},
	}.Execute()
	if err != nil {
		// do not throw error as it is just to warn users
		return
	}
	if len(l.SetterCounts) > 0 {
		fmt.Println("Warning: This package is using older version of setters which " +
			"will be deprecated in v0.38.0(expected release date is 11/25/2020) version of kpt, please " +
			"use 'kpt pkg fix -h' for instructions about upgrading it")
	}
}
