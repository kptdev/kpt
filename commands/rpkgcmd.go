// Copyright 2022 Google LLC
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
	"context"
	"flag"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/cmdlist"
	"github.com/GoogleContainerTools/kpt/internal/cmdres"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/alphadocs"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

func NewRpkgCommand(ctx context.Context, version string) *cobra.Command {
	repo := &cobra.Command{
		Use:     "rpkg",
		Aliases: []string{"rpackage"},
		Short:   alphadocs.RpkgShort,
		Long:    alphadocs.RpkgLong,
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
		Hidden: true,
	}

	pf := repo.PersistentFlags()

	kubeflags := genericclioptions.NewConfigFlags(true)
	kubeflags.AddFlags(pf)

	kubeflags.WrapConfigFn = func(rc *rest.Config) *rest.Config {
		rc.UserAgent = fmt.Sprintf("kpt/%s", version)
		return rc
	}

	pf.AddGoFlagSet(flag.CommandLine)

	repo.AddCommand(
		cmdlist.NewCommand(ctx, kubeflags),
		cmdres.NewCommand(ctx, kubeflags),
	)

	return repo
}
