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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func GetTTLCommand(name string) *cobra.Command {
	var speed float32
	var print bool
	ttl := &cobra.Command{
		Use:     "ttl",
		Short:   READMEShort,
		Long:    READMEShort + "\n" + READMELong,
		Example: READMEExamples,
		Aliases: []string{"tutorials", "tutorial"},
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := exec.LookPath("asciinema")
			if err != nil {
				fmt.Fprintln(os.Stderr, "must install asciinema to run tutorials: https://asciinema.org")
				os.Exit(1)
			}
			h, err := cmd.Flags().GetBool("help")
			if err != nil {
				return err
			}
			if h {
				return cmd.Help()
			}
			if len(args) == 0 {
				args = []string{"kpt"}
			}

			var c *exec.Cmd
			if print {
				c = exec.Command(p, "cat",
					fmt.Sprintf("https://storage.googleapis.com/kpt-dev/docs/%s.cast",
						strings.Join(args, "-")))
			} else {
				c = exec.Command(p, "play", "--speed", fmt.Sprintf("%f", speed),
					fmt.Sprintf("https://storage.googleapis.com/kpt-dev/docs/%s.cast",
						strings.Join(args, "-")))
			}
			c.Stdin = cmd.InOrStdin()
			c.Stdout = cmd.OutOrStdout()
			c.Stderr = cmd.ErrOrStderr()
			return c.Run()
		},
	}
	ttl.Flags().Float32VarP(
		&speed, "speed", "s", 1, "playback speedup (can be fractional)")
	ttl.Flags().BoolVar(
		&print, "print", false, "print the tutorial instead of playing it")

	return ttl
}

//TODO: Figure out if this can be generated from the site.

var READMEShort = `Run command tutorials`
var READMELong = `
Tutorials simulates running a sequence of commands and their output by playing
an asciinema cast.

Requires [asciinema].
`
var READMEExamples = `
    # run the tutorial for ` + "`" + `kpt cfg annotate` + "`" + `
    kpt ttl cfg annotate

    # run the tutorial at 2x speed
    kpt ttl cfg annotate -s 2

    # run the tutorial at 1/2x speed
    kpt ttl cfg annotate -s 0.5

    # print the full tutorial rather than playing it
    kpt ttl cfg annotate --print`
