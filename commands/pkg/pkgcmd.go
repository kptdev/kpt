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

package pkg

import (
	"context"

	"github.com/GoogleContainerTools/kpt/commands/pkg/diff"
	"github.com/GoogleContainerTools/kpt/commands/pkg/get"
	initialization "github.com/GoogleContainerTools/kpt/commands/pkg/init"
	"github.com/GoogleContainerTools/kpt/commands/pkg/update"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/cmdtree"
	"github.com/spf13/cobra"
)

func GetCommand(ctx context.Context, name string) *cobra.Command {
	pkg := &cobra.Command{
		Use:     "pkg",
		Short:   pkgdocs.PkgShort,
		Long:    pkgdocs.PkgLong,
		Aliases: []string{"package"},
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

	pkg.AddCommand(
		get.NewCommand(ctx, name), initialization.NewCommand(ctx, name),
		update.NewCommand(ctx, name), diff.NewCommand(ctx, name),
		cmdtree.NewCommand(ctx, name),
	)
	return pkg
}
