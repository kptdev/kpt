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

package license

import (
	"context"

	"github.com/GoogleContainerTools/kpt/commands/alpha/license/info"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/licensedocs"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, _ string) *cobra.Command {
	licenseCmd := &cobra.Command{
		Use:   "license",
		Short: "[Alpha] " + licensedocs.LicenseShort,
		Long:  "[Alpha] " + licensedocs.LicenseShort + "\n" + licensedocs.LicenseLong,
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

	licenseCmd.AddCommand(
		info.NewCommand(ctx),
	)

	return licenseCmd
}
