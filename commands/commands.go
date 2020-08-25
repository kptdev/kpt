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
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/cmddesc"
	"github.com/GoogleContainerTools/kpt/internal/cmddiff"
	"github.com/GoogleContainerTools/kpt/internal/cmdfix"
	"github.com/GoogleContainerTools/kpt/internal/cmdget"
	"github.com/GoogleContainerTools/kpt/internal/cmdinit"
	"github.com/GoogleContainerTools/kpt/internal/cmdsync"
	"github.com/GoogleContainerTools/kpt/internal/cmdupdate"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/cmd/util"
)

func GetAnthosCommands(name string) []*cobra.Command {
	c := []*cobra.Command{cmddesc.NewCommand(name),
		cmdget.NewCommand(name), cmdinit.NewCommand(name),
		cmdsync.NewCommand(name),
		cmdupdate.NewCommand(name), cmddiff.NewCommand(name),
		cmdfix.NewCommand(name),
	}

	// apply cross-cutting issues to commands
	NormalizeCommand(c...)
	return c
}

// NormalizeCommand will modify commands to be consistent, e.g. silencing errors
func NormalizeCommand(c ...*cobra.Command) {
	for i := range c {
		cmd := c[i]
		cmd.Short = strings.TrimPrefix(cmd.Short, "[Alpha] ")
		NormalizeCommand(cmd.Commands()...)
	}
}

// GetKptCommands returns the set of kpt commands to be registered
func GetKptCommands(name string, f util.Factory) []*cobra.Command {
	var c []*cobra.Command
	cfgCmd := GetConfigCommand(name)
	fnCmd := GetFnCommand(name)
	pkgCmd := GetPkgCommand(name)
	ttlCmd := GetTTLCommand(name)
	liveCmd := GetLiveCommand(name, f)
	guideCmd := GetGuideCommand(name)

	c = append(c, cfgCmd, pkgCmd, fnCmd, ttlCmd, liveCmd, guideCmd)

	// apply cross-cutting issues to commands
	NormalizeCommand(c...)
	return c
}
