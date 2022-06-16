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

	"github.com/GoogleContainerTools/kpt/internal/cmdrpkgapprove"
	"github.com/GoogleContainerTools/kpt/internal/cmdrpkgclone"
	"github.com/GoogleContainerTools/kpt/internal/cmdrpkgcopy"
	"github.com/GoogleContainerTools/kpt/internal/cmdrpkgdel"
	"github.com/GoogleContainerTools/kpt/internal/cmdrpkgget"
	"github.com/GoogleContainerTools/kpt/internal/cmdrpkginit"
	"github.com/GoogleContainerTools/kpt/internal/cmdrpkgpropose"
	"github.com/GoogleContainerTools/kpt/internal/cmdrpkgpull"
	"github.com/GoogleContainerTools/kpt/internal/cmdrpkgpush"
	"github.com/GoogleContainerTools/kpt/internal/cmdrpkgreject"
	"github.com/GoogleContainerTools/kpt/internal/cmdrpkgupdate"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/rpkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

func NewRpkgCommand(ctx context.Context, version string) *cobra.Command {
	repo := &cobra.Command{
		Use:     "rpkg",
		Aliases: []string{"rpackage"},
		Short:   "[Alpha] " + rpkgdocs.RpkgShort,
		Long:    "[Alpha] " + rpkgdocs.RpkgLong,
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
		Hidden: porch.HidePorchCommands,
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
		cmdrpkgget.NewCommand(ctx, kubeflags),
		cmdrpkgpull.NewCommand(ctx, kubeflags),
		cmdrpkgpush.NewCommand(ctx, kubeflags),
		cmdrpkgclone.NewCommand(ctx, kubeflags),
		cmdrpkginit.NewCommand(ctx, kubeflags),
		cmdrpkgpropose.NewCommand(ctx, kubeflags),
		cmdrpkgapprove.NewCommand(ctx, kubeflags),
		cmdrpkgreject.NewCommand(ctx, kubeflags),
		cmdrpkgdel.NewCommand(ctx, kubeflags),
		cmdrpkgcopy.NewCommand(ctx, kubeflags),
		cmdrpkgupdate.NewCommand(ctx, kubeflags),
	)

	return repo
}
