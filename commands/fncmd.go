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
	fndocs "github.com/GoogleContainerTools/kpt/internal/docs/generated/functions"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/cmd/config/configcobra"
)

func GetFnCommand(name string) *cobra.Command {
	functions := &cobra.Command{
		Use:   "functions",
		Short: "Generate and mutate local configuration by running functional images",
	}

	run := configcobra.RunFn(name)
	run.Short = fndocs.RunShort
	run.Long = fndocs.RunLong
	run.Example = fndocs.RunExamples

	source := configcobra.Source(name)
	source.Short = fndocs.SourceShort
	source.Long = fndocs.SourceLong
	source.Example = fndocs.SourceExamples

	sink := configcobra.Sink(name)
	sink.Short = fndocs.SinkShort
	sink.Long = fndocs.SinkLong
	sink.Example = fndocs.SinkExamples

	functions.AddCommand(run, source, sink)
	return functions
}
