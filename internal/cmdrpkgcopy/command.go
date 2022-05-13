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

package cmdrpkgcopy

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/rpkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkgcopy"
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
		Use:     "copy SOURCE_PACKAGE NAME",
		Aliases: []string{"edit"},
		Short:   rpkgdocs.CopyShort,
		Long:    rpkgdocs.CopyShort + "\n" + rpkgdocs.CopyLong,
		Example: rpkgdocs.CopyExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c
	// TODO (natasha41575): Make the default "latest+1"
	c.Flags().StringVar(&r.revision, "revision", "", "Revision of the copied package.")

	return r
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command

	copy porchapi.PackageEditTaskSpec

	revision string // Target package revision
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"
	client, err := porch.CreateClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client

	if len(args) < 1 {
		return errors.E(op, fmt.Errorf("SOURCE_PACKAGE is a required positional argument"))
	}
	if len(args) > 1 {
		return errors.E(op, fmt.Errorf("too many arguments; SOURCE_PACKAGE is the only accepted positional arguments"))
	}

	// TODO(natasha41575): This is temporarily required until we can set a default value for the revision. Now that we are disallowing
	//   package name changes or editing outside the repository, the copy needs to have a new revision number.
	if r.revision == "" {
		return errors.E(op, fmt.Errorf("--revision is a required flag"))
	}

	r.copy.Source = &porchapi.PackageRevisionRef{
		Name: args[0],
	}
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	revisionSpec, err := r.getPackageRevisionSpec()
	if err != nil {
		return errors.E(op, err)
	}
	pr := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: *r.cfg.Namespace,
		},
		Spec: *revisionSpec,
	}
	if err := r.client.Create(r.ctx, pr); err != nil {
		return errors.E(op, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s created", pr.Name)
	return nil
}

func (r *runner) getPackageRevisionSpec() (*porchapi.PackageRevisionSpec, error) {
	newScheme := runtime.NewScheme()
	if err := porchapi.SchemeBuilder.AddToScheme(newScheme); err != nil {
		return nil, err
	}
	restClient, err := porch.CreateRESTClient(r.cfg)
	if err != nil {
		return nil, err
	}

	result := porchapi.PackageRevision{}
	err = restClient.
		Get().
		Namespace(*r.cfg.Namespace).
		Resource("packagerevisions").
		Name(r.copy.Source.Name).
		Do(context.Background()).
		Into(&result)
	if err != nil {
		return nil, err
	}

	// TODO(natasha41575): Set a default revision of "latest + 1"
	spec := &porchapi.PackageRevisionSpec{
		PackageName:    result.Spec.PackageName,
		Revision:       r.revision,
		RepositoryName: result.Spec.RepositoryName,
	}
	spec.Tasks = []porchapi.Task{{Type: porchapi.TaskTypeEdit, Edit: &r.copy}}
	return spec, nil
}
