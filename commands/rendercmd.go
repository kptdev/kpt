// Copyright 2020 Google LLC
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
	"io"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
)

func GetRenderCommand(name string) *cobra.Command {
	rr := &RenderRunner{}

	cmd := &cobra.Command{
		Use: "render",
		// Short:   RenderShort,
		// Long:    RenderShort + "\n" + RenderLong,
		// Example: RenderExamples,
		// Aliases: []string{},
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := cmd.Flags().GetBool("help")
			if err != nil {
				return err
			}
			if h {
				return cmd.Help()
			}
			if len(args) == 0 {
				return fmt.Errorf("must pass path to DRY config")
			}

			rr.kustomizationPath = args[0]

			return rr.RunBuild(cmd.OutOrStdout())
		},
	}

	return cmd
}

type RenderRunner struct {
	kustomizationPath string
}

func (r *RenderRunner) makeOptions() *krusty.Options {
	opts := krusty.MakeDefaultOptions()
	opts.DoLegacyResourceSort = false
	opts.LoadRestrictions = types.LoadRestrictionsRootOnly
	opts.AddManagedbyLabel = false
	opts.UseKyaml = false
	return opts
}

func (r *RenderRunner) RunBuild(out io.Writer) error {
	fSys := filesys.MakeFsOnDisk()
	k := krusty.MakeKustomizer(fSys, r.makeOptions())
	m, err := k.Run(r.kustomizationPath)
	if err != nil {
		return err
	}
	return r.emitResources(out, fSys, m)
}

func (r *RenderRunner) emitResources(out io.Writer, _ filesys.FileSystem, m resmap.ResMap) error {
	res, err := m.AsYaml()
	if err != nil {
		return err
	}
	_, err = out.Write(res)
	return err
}
