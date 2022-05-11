// Copyright 2020 Google LLC
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
	"flag"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/util/cfgflags"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cluster "k8s.io/kubectl/pkg/cmd/util"
)

func newFactory(cmd *cobra.Command, version string) cluster.Factory {
	flags := cmd.PersistentFlags()
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.AddFlags(flags)
	userAgentKubeConfigFlags := &cfgflags.UserAgentKubeConfigFlags{
		Delegate:  kubeConfigFlags,
		UserAgent: fmt.Sprintf("kpt/%s", version),
	}
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return cluster.NewFactory(userAgentKubeConfigFlags)
}
