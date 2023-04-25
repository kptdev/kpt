// Copyright 2022 The kpt Authors
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

package alpha

import (
	"context"

	"github.com/GoogleContainerTools/kpt/commands/alpha/license"
	"github.com/GoogleContainerTools/kpt/commands/alpha/live"
	"github.com/GoogleContainerTools/kpt/commands/alpha/repo"
	"github.com/GoogleContainerTools/kpt/commands/alpha/rollouts"
	"github.com/GoogleContainerTools/kpt/commands/alpha/rpkg"
	"github.com/GoogleContainerTools/kpt/commands/alpha/sync"
	"github.com/GoogleContainerTools/kpt/commands/alpha/wasm"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/alphadocs"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/spf13/cobra"
)

func GetCommand(ctx context.Context, _, version string) *cobra.Command {
	alpha := &cobra.Command{
		Use:   "alpha",
		Short: alphadocs.AlphaShort,
		Long:  alphadocs.AlphaLong,
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

	alpha.AddCommand(
		repo.NewCommand(ctx, version),
		rpkg.NewCommand(ctx, version),
		sync.NewCommand(ctx, version),
		wasm.NewCommand(ctx, version),
		live.GetCommand(ctx, "", version),
		license.NewCommand(ctx, version),
		rollouts.NewCommand(ctx, version),
	)

	return alpha
}
