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

package commands

import (
	"context"
	"strings"

	"github.com/GoogleContainerTools/kpt/commands/alpha"
	"github.com/GoogleContainerTools/kpt/commands/fn"
	"github.com/GoogleContainerTools/kpt/commands/live"
	"github.com/GoogleContainerTools/kpt/commands/pkg"
	"github.com/spf13/cobra"
)

// NormalizeCommand will modify commands to be consistent, e.g. silencing errors
func NormalizeCommand(c ...*cobra.Command) {
	for _, cmd := range c {
		cmd.Short = strings.TrimPrefix(cmd.Short, "[Alpha] ")
		NormalizeCommand(cmd.Commands()...)
	}
}

// GetKptCommands returns the set of kpt commands to be registered
func GetKptCommands(ctx context.Context, name, version string) []*cobra.Command {
	var c []*cobra.Command
	fnCmd := fn.GetCommand(ctx, name)
	pkgCmd := pkg.GetCommand(ctx, name)
	liveCmd := live.GetCommand(ctx, name, version)
	alphaCmd := alpha.GetCommand(ctx, name, version)

	c = append(c, pkgCmd, fnCmd, liveCmd, alphaCmd)

	// apply cross-cutting issues to commands
	NormalizeCommand(c...)
	return c
}
