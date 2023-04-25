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

package proposedelete

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/rpkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkgpropose-delete"
)

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	c := &cobra.Command{
		Use:        "propose-delete PACKAGE",
		Aliases:    []string{"propose-del"},
		Short:      rpkgdocs.ProposeDeleteShort,
		Long:       rpkgdocs.ProposeDeleteShort + "\n" + rpkgdocs.ProposeDeleteLong,
		Example:    rpkgdocs.ProposeDeleteExamples,
		SuggestFor: []string{},
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

func (r *runner) preRunE(_ *cobra.Command, _ []string) error {
	const op errors.Op = command + ".preRunE"

	client, err := porch.CreateClientWithFlags(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client
	return nil
}

func (r *runner) runE(_ *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"
	var messages []string
	namespace := *r.cfg.Namespace

	for _, name := range args {
		pr := &v1alpha1.PackageRevision{}
		if err := r.client.Get(r.ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, pr); err != nil {
			return errors.E(op, err)
		}

		switch pr.Spec.Lifecycle {
		case v1alpha1.PackageRevisionLifecyclePublished:
			// ok
		case v1alpha1.PackageRevisionLifecycleDeletionProposed:
			fmt.Fprintf(r.Command.OutOrStderr(), "%s is already proposed for deletion\n", name)
			continue
		default:
			msg := fmt.Sprintf("can only propose published packages for deletion; package %s is not published", name)
			messages = append(messages, msg)
			fmt.Fprintln(r.Command.ErrOrStderr(), msg)
			continue
		}

		pr.Spec.Lifecycle = v1alpha1.PackageRevisionLifecycleDeletionProposed
		if err := r.client.Update(r.ctx, pr); err != nil {
			messages = append(messages, err.Error())
			fmt.Fprintf(r.Command.ErrOrStderr(), "%s failed (%s)\n", name, err)
		} else {
			fmt.Fprintf(r.Command.OutOrStderr(), "%s proposed for deletion\n", name)
		}
	}

	if len(messages) > 0 {
		return errors.E(op, fmt.Errorf("errors:\n  %s", strings.Join(messages, "\n  ")))
	}

	return nil
}
