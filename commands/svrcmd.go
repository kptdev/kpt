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
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/svrdocs"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/cmd/kubectl/kubectlcobra"
	"sigs.k8s.io/kustomize/cmd/resource/status"
)

func GetSvrCommand(name string) *cobra.Command {
	cluster := &cobra.Command{
		Use:     "svr",
		Short:   svrdocs.READMEShort,
		Long:    svrdocs.READMEShort + "\n" + svrdocs.READMELong,
		Example: svrdocs.READMEExamples,
		Aliases: []string{"server"},
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
	cluster.AddCommand(status.StatusCommand())
	cluster.AddCommand(kubectlcobra.GetCommand(nil).Commands()...)
	return cluster
}
