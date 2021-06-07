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

package run

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	kptcommands "github.com/GoogleContainerTools/kpt/commands"
	"github.com/GoogleContainerTools/kpt/internal/cmdcomplete"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/overview"
	"github.com/GoogleContainerTools/kpt/internal/util/cfgflags"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	kptopenapi "github.com/GoogleContainerTools/kpt/internal/util/openapi"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/util/factory"
	"sigs.k8s.io/kustomize/cmd/config/ext"
	"sigs.k8s.io/kustomize/kyaml/commandutil"
	"sigs.k8s.io/kustomize/kyaml/pathutil"
)

var pgr []string

func GetMain() *cobra.Command {
	os.Setenv(commandutil.EnableAlphaCommmandsEnvName, "true")
	installComp := false
	cmd := &cobra.Command{
		Use:          "kpt",
		Short:        overview.ReferenceShort,
		Long:         overview.ReferenceLong,
		Example:      overview.ReferenceExamples,
		SilenceUsage: true,
		// We handle all errors in main after return from cobra so we can
		// adjust the error message coming from libraries
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if installComp {
				os.Setenv("COMP_INSTALL", "1")
				os.Setenv("COMP_YES", "1")
				fmt.Fprint(cmd.OutOrStdout(), "Installing shell completion...\n")
				fmt.Fprint(cmd.OutOrStdout(),
					"This will add 'complete -C /Users/$USER/go/bin/kpt kpt' to "+
						".bashrc, .bash_profile, etc\n")
				fmt.Fprint(cmd.OutOrStdout(), "Run `COMP_INSTALL=0 kpt` to uninstall.\n")
			}
			// Complete exits if it is called in completion mode, otherwise it is a no-op
			cmdcomplete.Complete(cmd, false, nil).Complete("kpt")

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

	f := newFactory(cmd)

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// register function to use Kptfile for OpenAPI
		ext.KRMFileName = func() string {
			return kptfile.KptFileName
		}
		err := kptopenapi.ConfigureOpenAPI(f, cmdutil.K8sSchemaSource, cmdutil.K8sSchemaPath)
		if err != nil {
			return err
		}

		if err := verifyKptfileVersion(cmd, args); err != nil {
			return err
		}

		return nil
	}

	cmd.Flags().BoolVar(&installComp, "install-completion", false,
		"install shell completion")
	// this command will be invoked by the shell-completion code
	cmd.AddCommand(&cobra.Command{
		Use:           "kpt",
		Hidden:        true,
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, args []string) {
			// Complete exits if it is called in completion mode, otherwise it is a no-op
			cmdcomplete.Complete(cmd.Parent(), false, nil).Complete("kpt")
		},
	})

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
	cmd.AddCommand(kptcommands.GetKptCommands("kpt", f)...)

	// enable stack traces
	cmd.PersistentFlags().BoolVar(&cmdutil.StackOnError, "stack-trace", false,
		"print a stack-trace on failure")

	cmd.PersistentFlags().StringVar(&cmdutil.K8sSchemaSource, "k8s-schema-source",
		kptopenapi.SchemaSourceBuiltin, "source for the kubernetes openAPI schema")
	cmd.PersistentFlags().StringVar(&cmdutil.K8sSchemaPath, "k8s-schema-path",
		"./openapi.json", "path to the kubernetes openAPI schema file")

	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintf(os.Stderr, "kpt requires that `git` is installed and on the PATH")
		os.Exit(1)
	}

	replace(cmd)

	cmd.AddCommand(versionCmd)
	hideFlags(cmd)
	return cmd
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

func newFactory(cmd *cobra.Command) util.Factory {
	flags := cmd.PersistentFlags()
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.AddFlags(flags)
	userAgentKubeConfigFlags := &cfgflags.UserAgentKubeConfigFlags{
		Delegate:  kubeConfigFlags,
		UserAgent: fmt.Sprintf("kpt/%s", version),
	}
	matchVersionKubeConfigFlags := util.NewMatchVersionFlags(
		&factory.CachingRESTClientGetter{
			Delegate: userAgentKubeConfigFlags,
		},
	)
	matchVersionKubeConfigFlags.AddFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return util.NewFactory(matchVersionKubeConfigFlags)
}

var version = "unknown"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of kpt",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s\n", version)
	},
}

// hideFlags hides any cobra flags that are unlikely to be used by
// customers.
func hideFlags(cmd *cobra.Command) {
	flags := []string{
		// Flags related to logging
		"add_dir_header",
		"alsologtostderr",
		"log_backtrace_at",
		"log_dir",
		"log_file",
		"log_file_max_size",
		"skip_headers",
		"skip_log_headers",
		"vmodule",

		// Flags related to apiserver
		"as",
		"as-group",
		"cache-dir",
		"certificate-authority",
		"client-certificate",
		"client-key",
		"insecure-skip-tls-verify",
		"match-server-version",
		"password",
		"token",
		"username",
	}
	for _, f := range flags {
		_ = cmd.PersistentFlags().MarkHidden(f)
	}
}

// verifyKptfileVersion checks whether the DIR arg is provided, and if so,
// checks if any Kptfiles has the correct GVK.
func verifyKptfileVersion(cmd *cobra.Command, args []string) error {
	var cmdChain []string
	c := cmd
	for {
		cmdChain = append([]string{c.Name()}, cmdChain...)
		c = c.Parent()
		if c == nil {
			break
		}
	}

	// If the user just used "$ kpt" without any subcommand, just do nothing.
	if len(cmdChain) < 2 {
		return nil
	}

	// The handling here depends on the command group.
	switch cmdChain[1] {
	// For commands that doesn't take the path to a Kptfile as an argument we
	// don't need to check anything.
	case "version":
		fallthrough
	case "ttl":
		fallthrough
	case "help":
		fallthrough
	case "guide":
		return nil

	// The pkg commands needs special handling and the code is all in the kpt
	// repo. So no need to do a check here.
	case "pkg":
		return nil
	}

	// For the cfg, fn and live command groups, we need to verify the
	// version of the Kptfile schema here.
	if len(args) == 0 {
		// If there are no arguments we don't have a directory where we can
		// look for Kptfiles, so just return.
		return nil
	}

	pathArg := args[0]
	var fullPath string
	if filepath.IsAbs(pathArg) {
		fullPath = pathArg
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		fullPath = filepath.Join(cwd, pathArg)
	}

	_, err := os.Stat(fullPath)
	if err != nil {
		// If the folder doesn't exist, we don't do any checks here and rely
		// on other functionality to report any errors.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	paths, err := pathutil.DirsWithFile(fullPath, kptfile.KptFileName, true)
	if err != nil {
		return err
	}

	for _, p := range paths {
		_, err := kptfileutil.ReadFile(p)
		if err != nil {
			var unknownKptfileVersionError *kptfileutil.UnknownKptfileVersionError
			if errors.As(err, &unknownKptfileVersionError) {
				unknownKptfileVersionError.PkgPath = p
			}
			return err
		}
	}
	return nil
}
