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

package info

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/licenses"
	"github.com/spf13/cobra"
)

func newRunner(ctx context.Context) *runner {
	r := &runner{
		ctx: ctx,
	}
	c := &cobra.Command{
		Use:   "info",
		Short: "Displays licenses for the OSS libraries used by kpt CLI.",
		RunE:  r.runE,
	}
	r.Command = c
	return r
}

func NewCommand(ctx context.Context) *cobra.Command {
	return newRunner(ctx).Command
}

type runner struct {
	ctx     context.Context
	Command *cobra.Command
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	fmt.Println(licenses.AllOSSLicense)
	return nil
}
