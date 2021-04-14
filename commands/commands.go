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
	"context"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/cmd/util"
)

// NormalizeCommand will modify commands to be consistent, e.g. silencing errors
func NormalizeCommand(c ...*cobra.Command) {
	for _, cmd := range c {
		cmd.Short = strings.TrimPrefix(cmd.Short, "[Alpha] ")
		NormalizeCommand(cmd.Commands()...)
	}
}

// GetKptCommands returns the set of kpt commands to be registered
func GetKptCommands(ctx context.Context, name string, f util.Factory) []*cobra.Command {
	var c []*cobra.Command
	fnCmd := GetFnCommand(ctx, name)
	pkgCmd := GetPkgCommand(name)
	liveCmd := GetLiveCommand(name, f)

	c = append(c, pkgCmd, fnCmd, liveCmd)

	// apply cross-cutting issues to commands
	NormalizeCommand(c...)
	return c
}
