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

// Package cmdcomplete contains the completion command
package cmdcomplete

import (
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/util/update"
	"github.com/posener/complete/v2"
	"github.com/posener/complete/v2/predict"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type VisitFlags func(cmd *cobra.Command, flag *pflag.Flag, cc *complete.Command)

// Complete returns a completion command for a cobra command
func Complete(cmd *cobra.Command, skipHelp bool, visitFlags VisitFlags) *complete.Command {
	cc := &complete.Command{
		Flags: map[string]complete.Predictor{},
		Sub:   map[string]*complete.Command{},
	}
	if strings.Contains(cmd.Use, "DIR") {
		// if usage contains directory, then use a file predictor
		cc.Args = predict.Dirs("*")
	}

	// add each command
	if !skipHelp {
		cc.Sub["help"] = &complete.Command{Sub: map[string]*complete.Command{}}
	}
	for i := range cmd.Commands() {
		c := cmd.Commands()[i]
		if c.Hidden || c.Deprecated != "" {
			continue
		}
		name := strings.Split(c.Use, " ")[0]
		cc.Sub[name] = Complete(c, true, visitFlags)
		if !skipHelp {
			cc.Sub["help"].Sub[name] = cc.Sub[name]
		}
	}

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if visitFlags != nil {
			// extension support for other commands that embed this one
			visitFlags(cmd, flag, cc)
		}
		if flag.Name == "strategy" {
			cc.Flags[flag.Name] = predict.Options(predict.OptValues(update.Strategies...))
			return
		}
		if flag.Name == "pattern" {
			cc.Flags[flag.Name] = predict.Options(predict.OptValues("%k_%n.yaml"))
			return
		}
		cc.Flags[flag.Name] = predict.Nothing
	})

	return cc
}
