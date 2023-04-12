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

package doc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"github.com/spf13/cobra"
)

func NewRunner(ctx context.Context, parent string) *Runner {
	r := &Runner{
		Ctx: ctx,
	}
	c := &cobra.Command{
		Use:     "doc --image=IMAGE",
		Args:    cobra.MaximumNArgs(0),
		Short:   fndocs.DocShort,
		Long:    fndocs.DocShort + "\n" + fndocs.DocLong,
		Example: fndocs.DocExamples,
		RunE:    r.runE,
	}
	r.Command = c
	c.Flags().StringVarP(&r.Image, "image", "i", "", "kpt function image name")
	_ = r.Command.RegisterFlagCompletionFunc("image", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.SuggestFunctions(cmd), cobra.ShellCompDirectiveDefault
	})
	cmdutil.FixDocs("kpt", parent, c)
	return r
}

func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).Command
}

type Runner struct {
	Image   string
	Command *cobra.Command
	Ctx     context.Context
}

func (r *Runner) runE(c *cobra.Command, _ []string) error {
	if r.Image == "" {
		return errors.New("image must be specified")
	}
	// TODO: We probably should be going through the runner
	image, err := fnruntime.ResolveToImageForCLI(c.Context(), r.Image)
	if err != nil {
		return err
	}
	var out, errout bytes.Buffer
	dockerRunArgs := []string{
		"run",
		"--rm", // delete the container afterward
		image,
		"--help",
	}
	// If the env var is empty, stringToContainerRuntime defaults it to docker.
	runtime, err := fnruntime.StringToContainerRuntime(os.Getenv(fnruntime.ContainerRuntimeEnv))
	if err != nil {
		return err
	}

	err = fnruntime.ContainerRuntimeAvailable(runtime)
	if err != nil {
		return err
	}

	cmd := exec.Command(runtime.GetBin(), dockerRunArgs...)
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err = cmd.Run()
	pr := printer.FromContextOrDie(r.Ctx)
	if err != nil {
		pr.Printf(errout.String())
		return fmt.Errorf("please ensure the container has an entrypoint and it supports --help flag: %w", err)
	}
	fmt.Fprintln(pr.OutStream(), out.String())
	return nil
}
