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
	"github.com/GoogleContainerTools/kpt/internal/cmdexport"
	"github.com/GoogleContainerTools/kpt/internal/cmdfndoc"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/GoogleContainerTools/kpt/internal/pipeline"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/cmdeval"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/cmdsink"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/cmdsource"
	"github.com/spf13/cobra"
)

func GetFnCommand(name string) *cobra.Command {
	functions := &cobra.Command{
		Use:     "fn",
		Short:   fndocs.FnShort,
		Long:    fndocs.FnLong,
		Example: fndocs.FnExamples,
		Aliases: []string{"functions"},
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

	eval := cmdeval.EvalCommand(name)
	eval.Short = fndocs.RunShort
	eval.Long = fndocs.RunShort + "\n" + fndocs.RunLong
	eval.Example = fndocs.RunExamples

	render := pipeline.NewCommand(name)

	doc := cmdfndoc.NewCommand(name)
	doc.Short = fndocs.DocShort
	doc.Long = fndocs.DocShort + "\n" + fndocs.DocLong
	doc.Example = fndocs.DocExamples

	source := cmdsource.NewCommand(name)

	sink := cmdsink.NewCommand(name)

	functions.AddCommand(eval, render, doc, source, sink, cmdexport.ExportCommand())
	return functions
}
