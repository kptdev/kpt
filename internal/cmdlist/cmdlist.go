// Copyright 2022 Google LLC
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

package cmdlist

import (
	"context"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const listLong string = `
kpt alpha rpkg list [flags]

Flags:

--name
	Name of the packages to list. Any package whose name contains this value will be included in the results.

`

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	c := &cobra.Command{
		Use:        "list",
		Aliases:    []string{},
		SuggestFor: []string{},
		Short:      "Lists packages in registered repositories.",
		Long:       listLong,
		Example:    "TODO",
		PreRunE:    r.preRunE,
		RunE:       r.runE,
		Hidden:     true,
	}
	r.Command = c

	c.Flags().StringVar(&r.name, "name", "", "Name of the packages to list. Any package whose name contains this value will be included in the results.")

	return r
}

func NewCommand(ctx context.Context, rcg *genericclioptions.ConfigFlags) *cobra.Command {
	return newRunner(ctx, rcg).Command
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command

	printer printer.Printer

	// Flags
	name string
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = "cmdlist.preRunE"
	config, err := r.cfg.ToRESTConfig()
	if err != nil {
		return errors.E(op, err)
	}

	scheme, err := createScheme()
	if err != nil {
		return errors.E(op, err)
	}

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return errors.E(op, err)
	}

	r.client = c
	r.printer = printer.FromContextOrDie(r.ctx)
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = "cmdlist.runE"

	var list porchapi.PackageRevisionList
	if err := r.client.List(r.ctx, &list); err != nil {
		return errors.E(op, err)
	}

	// TODO: server-side filtering

	for i := range list.Items {
		pr := &list.Items[i]
		if r.match(pr) {
			r.printer.Printf("%s/%s   %s\n", pr.Namespace, pr.Name, pr.Spec.PackageName)
		}
	}

	return nil
}

func (r *runner) match(pr *porchapi.PackageRevision) bool {
	return strings.Contains(pr.Name, r.name)
}

func createScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	for _, api := range (runtime.SchemeBuilder{
		porchapi.AddToScheme,
		configapi.AddToScheme,
	}) {
		if err := api(scheme); err != nil {
			return nil, err
		}
	}
	return scheme, nil
}
