// Copyright 2019 The kpt Authors
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

package fn

import (
	"context"

	"github.com/GoogleContainerTools/kpt/commands/fn/doc"
	"github.com/GoogleContainerTools/kpt/commands/fn/render"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/cmdeval"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/cmdsink"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/cmdsource"
	"github.com/spf13/cobra"
)

func GetCommand(ctx context.Context, name string) *cobra.Command {
	functions := &cobra.Command{
		Use:     "fn",
		Short:   fndocs.FnShort,
		Long:    fndocs.FnLong,
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

	functions.AddCommand(
		cmdeval.EvalCommand(ctx, name),
		render.NewCommand(ctx, name),
		doc.NewCommand(ctx, name),
		cmdsource.NewCommand(ctx, name),
		cmdsink.NewCommand(ctx, name),
	)
	return functions
}
