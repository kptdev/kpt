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

package cmdrpkgdel

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkgdel"
	longMsg = `
kpt alpha rpkg del[ete] [PACKAGE ...] [flags]

Args:

PACKAGE:
  Name of the package revision to delete.
`
)

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	c := &cobra.Command{
		Use:        "del",
		Aliases:    []string{"delete"},
		SuggestFor: []string{},
		Short:      "Deletes one or more packages in registered repositories.",
		Long:       longMsg,
		Example:    "kpt alpha rpkg del repository:package:v1 --namespace=default",
		PreRunE:    r.preRunE,
		RunE:       r.runE,
		Hidden:     porch.HidePorchCommands,
	}
	r.Command = c

	// Create flags

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
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"

	client, err := porch.CreateClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"
	var messages []string

	for _, pkg := range args {
		pr := &porchapi.PackageRevision{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PackageRevision",
				APIVersion: porchapi.SchemeGroupVersion.Identifier(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: *r.cfg.Namespace,
				Name:      pkg,
			},
		}

		switch err := r.client.Delete(r.ctx, pr); err {
		case nil:
			fmt.Fprintf(r.Command.ErrOrStderr(), "%s deleted", pkg)
		default:
			messages = append(messages, err.Error())
			fmt.Fprintf(r.Command.ErrOrStderr(), "%s failed (%s)", pkg, err)
		}
	}

	if len(messages) > 0 {
		return errors.E(op, fmt.Errorf("errors:\n  %s", strings.Join(messages, "\n  ")))
	}

	return nil
}
