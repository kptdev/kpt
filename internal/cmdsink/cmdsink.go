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

package cmdsink

import (
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/GoogleContainerTools/kpt/internal/runner"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

// GetSinkRunner returns a command for Sink.
func GetSinkRunner(name string) *SinkRunner {
	r := &SinkRunner{}
	c := &cobra.Command{
		Use:     "sink DIR",
		Short:   fndocs.SinkShort,
		Long:    fndocs.SinkLong,
		Example: fndocs.SinkExamples,
		RunE:    r.runE,
		Args:    cobra.MaximumNArgs(1),
	}
	r.Command = c
	return r
}

func NewCommand(name string) *cobra.Command {
	return GetSinkRunner(name).Command
}

// SinkRunner contains the run function
type SinkRunner struct {
	Command *cobra.Command
}

func (r *SinkRunner) runE(c *cobra.Command, args []string) error {
	var outputs []kio.Writer
	if len(args) == 1 {
		outputs = []kio.Writer{&kio.LocalPackageWriter{PackagePath: args[0]}}
	} else {
		outputs = []kio.Writer{&kio.ByteWriter{
			Writer:           c.OutOrStdout(),
			ClearAnnotations: []string{kioutil.PathAnnotation}},
		}
	}

	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: c.InOrStdin()}},
		Outputs: outputs}.Execute()
	return runner.HandleError(c, err)
}
