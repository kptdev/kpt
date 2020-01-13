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

//go:generate $GOBIN/mdtogo docs/tutorials internal/docs/generated/tutorials --full=true --license=none
//go:generate $GOBIN/mdtogo docs/pkg internal/docs/generated/pkg --license=none
//go:generate $GOBIN/mdtogo docs/config internal/docs/generated/config --license=none
//go:generate $GOBIN/mdtogo docs internal/docs/generated/overview --license=none
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/GoogleContainerTools/kpt/commands"
	"github.com/GoogleContainerTools/kpt/internal/cmdcomplete"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/overview"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/commandutil"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var pgr []string

func main() {
	os.Setenv(commandutil.EnableAlphaCommmandsEnvName, "true")
	cmd := &cobra.Command{
		Use:   "kpt",
		Short: overview.READMEShort,
		Long:  overview.READMELong,
		Example: `  #
  # get a package
  #
  $ export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git
  $ kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.1.0 helloworld
  fetching package /package-examples/helloworld-set from \
    git@github.com:GoogleContainerTools/kpt to helloworld

  #
  # list setters
  #
  $ kpt config list-setters helloworld
  NAME            DESCRIPTION         VALUE    TYPE     COUNT   SETBY
  http-port   'helloworld port'         80      integer   3
  image-tag   'hello-world image tag'   0.1.0   string    1
  replicas    'helloworld replicas'     5       integer   1

  #
  # set a value
  #
  $ kpt config set helloworld replicas 3 --set-by pwittrock \
    --description '3 is good enough'
  set 1 fields

  #
  # apply
  #
  $ kpt http apply -f helloworld
  deployment.apps/helloworld-gke created
  service/helloworld-gke created
`,
	}

	// find the pager if one exists
	func() {
		if val, found := os.LookupEnv("KPT_NO_PAGER_HELP"); !found || val != "1" {
			// use a pager for printing tutorials
			e, found := os.LookupEnv("PAGER")
			var err error
			if found {
				pgr = []string{e}
				return
			}
			e, err = exec.LookPath("pager")
			if err == nil {
				pgr = []string{e}
				return
			}
			e, err = exec.LookPath("less")
			if err == nil {
				pgr = []string{e, "-R"}
				return
			}
		}
	}()

	// help and documentation
	cmd.InitDefaultHelpCmd()
	cmd.AddCommand(commands.GetKptCommands("kpt")...)

	// enable stack traces
	cmd.PersistentFlags().BoolVar(&cmdutil.StackOnError, "stack-trace", false,
		"print a stack-trace on failure")

	// Complete exits if it is called in completion mode, otherwise it is a no-op
	cmdcomplete.Complete(cmd, false, nil).Complete("kpt")

	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintf(os.Stderr, "kpt requires that `git` is installed and on the PATH")
		os.Exit(1)
	}

	// exit on an error
	cmdutil.ExitOnError = true

	replace(cmd)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func replace(c *cobra.Command) {
	for i := range c.Commands() {
		replace(c.Commands()[i])
	}
	c.SetHelpFunc(newHelp(pgr, c))
}

func newHelp(e []string, c *cobra.Command) func(command *cobra.Command, strings []string) {
	if len(pgr) == 0 {
		return c.HelpFunc()
	}

	fn := c.HelpFunc()
	return func(command *cobra.Command, args []string) {
		stty := exec.Command("stty", "size")
		stty.Stdin = os.Stdin
		out, err := stty.Output()
		if err == nil {
			terminalHeight, err := strconv.Atoi(strings.Split(string(out), " ")[0])
			helpHeight := strings.Count(command.Long, "\n") +
				strings.Count(command.UsageString(), "\n")
			if err == nil && terminalHeight > helpHeight {
				// don't use a pager if the help is shorter than the console
				fn(command, args)
				return
			}
		}

		b := &bytes.Buffer{}
		pager := exec.Command(e[0])
		if len(e) > 1 {
			pager.Args = append(pager.Args, e[1:]...)
		}
		pager.Stdin = b
		pager.Stdout = c.OutOrStdout()
		c.SetOut(b)
		fn(command, args)
		if err := pager.Run(); err != nil {
			fmt.Fprintf(c.ErrOrStderr(), "%v", err)
			os.Exit(1)
		}
	}
}
