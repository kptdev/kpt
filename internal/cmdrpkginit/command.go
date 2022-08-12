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

package cmdrpkginit

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/rpkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkginit"
)

func NewCommand(ctx context.Context, rcg *genericclioptions.ConfigFlags) *cobra.Command {
	return newRunner(ctx, rcg).Command
}

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	c := &cobra.Command{
		Use:     "init PACKAGE_NAME",
		Short:   rpkgdocs.InitShort,
		Long:    rpkgdocs.InitShort + "\n" + rpkgdocs.InitLong,
		Example: rpkgdocs.InitExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	c.Flags().StringVar(&r.Description, "description", "sample description", "short description of the package.")
	c.Flags().StringSliceVar(&r.Keywords, "keywords", []string{}, "list of keywords for the package.")
	c.Flags().StringVar(&r.Site, "site", "", "link to page with information about the package.")
	c.Flags().StringVar(&r.repository, "repository", "", "Repository to which package will be cloned (downstream repository).")
	c.Flags().StringVar(&r.revision, "revision", "v1", "Revision of the downstream package.")

	return r
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command

	// Flags
	Keywords    []string
	Description string
	Site        string
	name        string // Target package name
	repository  string // Target repository
	revision    string // Target revision
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"

	client, err := porch.CreateClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client

	if len(args) < 1 {
		return errors.E(op, "PACKAGE_NAME is a required positional argument")
	}

	r.name = args[0]

	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: *r.cfg.Namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    r.name,
			Revision:       r.revision,
			RepositoryName: r.repository,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeInit,
					Init: &porchapi.PackageInitTaskSpec{
						Description: r.Description,
						Keywords:    r.Keywords,
						Site:        r.Site,
					},
				},
			},
		},
		Status: porchapi.PackageRevisionStatus{},
	}
	if err := r.client.Create(r.ctx, pr); err != nil {
		return errors.E(op, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s created\n", pr.Name)
	return nil
}
