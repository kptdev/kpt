// Copyright 2022 The kpt Authors
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

package get

import (
	"context"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/repodocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/options"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/get"
)

const (
	command = "cmdrepoget"
)

func NewCommand(ctx context.Context, rcg *genericclioptions.ConfigFlags) *cobra.Command {
	return newRunner(ctx, rcg).Command
}

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx:        ctx,
		getFlags:   options.Get{ConfigFlags: rcg},
		printFlags: get.NewGetPrintFlags(),
	}
	c := &cobra.Command{
		Use:     "get [REPOSITORY_NAME]",
		Aliases: []string{"ls", "list"},
		Short:   repodocs.GetShort,
		Long:    repodocs.GetShort + "\n" + repodocs.GetLong,
		Example: repodocs.GetExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	// Create flags
	r.getFlags.AddFlags(c)
	r.printFlags.AddFlags(c)
	return r
}

type runner struct {
	ctx     context.Context
	Command *cobra.Command

	// Flags
	getFlags   options.Get
	printFlags *get.PrintFlags

	requestTable bool
}

func (r *runner) preRunE(cmd *cobra.Command, _ []string) error {
	outputOption := cmd.Flags().Lookup("output").Value.String()
	if strings.Contains(outputOption, "custom-columns") || outputOption == "yaml" || strings.Contains(outputOption, "json") {
		r.requestTable = false
	} else {
		r.requestTable = true
	}
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	// For some reason our use of k8s libraries result in error when decoding
	// RepositoryList when we use strongly typed data. Therefore for now we
	// use unstructured communication.
	// The error is: `no kind "RepositoryList" is registered for the internal
	// version of group "config.porch.kpt.dev" in scheme`. Of course there _is_
	// no such kind since CRDs seem to have only versioned resources.
	b, err := r.getFlags.ResourceBuilder()
	if err != nil {
		return err
	}

	// TODO: Support table mode over proto
	// TODO: Print namespace in multi-namespace mode
	b = b.Unstructured()

	if len(args) > 0 {
		b.ResourceNames("repository", args...)
	} else {
		b = b.SelectAllParam(true).
			ResourceTypes("repository")
	}

	b = b.ContinueOnError().Latest().Flatten()

	if r.requestTable {
		b = b.TransformRequests(func(req *rest.Request) {
			req.SetHeader("Accept", strings.Join([]string{
				"application/json;as=Table;g=meta.k8s.io;v=v1",
				"application/json",
			}, ","))
		})
	}
	res := b.Do()
	if err := res.Err(); err != nil {
		return errors.E(op, err)
	}

	infos, err := res.Infos()
	if err != nil {
		return errors.E(op, err)
	}

	printer, err := r.printFlags.ToPrinter()
	if err != nil {
		return errors.E(op, err)
	}

	if r.requestTable {
		printer = &get.TablePrinter{
			Delegate: printer,
		}
	}

	w := printers.GetNewTabWriter(cmd.OutOrStdout())

	for _, i := range infos {
		if err := printer.PrintObj(i.Object, w); err != nil {
			return errors.E(op, err)
		}
	}

	if err := w.Flush(); err != nil {
		return errors.E(op, err)
	}

	return nil
}
