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

package reject

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
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkgreject"
)

func NewCommand(ctx context.Context, rcg *genericclioptions.ConfigFlags) *cobra.Command {
	return newRunner(ctx, rcg).Command
}

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx:    ctx,
		cfg:    rcg,
		client: nil,
	}

	c := &cobra.Command{
		Use:     "reject PACKAGE",
		Short:   rpkgdocs.RejectShort,
		Long:    rpkgdocs.RejectShort + "\n" + rpkgdocs.RejectLong,
		Example: rpkgdocs.RejectExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	return r
}

type runner struct {
	ctx         context.Context
	cfg         *genericclioptions.ConfigFlags
	client      rest.Interface
	porchClient client.Client
	Command     *cobra.Command

	// Flags
}

func (r *runner) preRunE(_ *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"

	if len(args) < 1 {
		return errors.E(op, "PACKAGE_REVISION is a required positional argument")
	}

	client, err := porch.CreateRESTClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client

	porchClient, err := porch.CreateClientWithFlags(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.porchClient = porchClient
	return nil
}

func (r *runner) runE(_ *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"
	var messages []string

	namespace := *r.cfg.Namespace

	for _, name := range args {
		pr := &v1alpha1.PackageRevision{}
		if err := r.porchClient.Get(r.ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, pr); err != nil {
			return errors.E(op, err)
		}
		switch pr.Spec.Lifecycle {
		case v1alpha1.PackageRevisionLifecycleProposed:
			if err := porch.UpdatePackageRevisionApproval(r.ctx, r.client, client.ObjectKey{
				Namespace: namespace,
				Name:      name,
			}, v1alpha1.PackageRevisionLifecycleDraft); err != nil {
				messages = append(messages, err.Error())
				fmt.Fprintf(r.Command.ErrOrStderr(), "%s failed (%s)\n", name, err)
			} else {
				fmt.Fprintf(r.Command.OutOrStderr(), "%s rejected\n", name)
			}
		case v1alpha1.PackageRevisionLifecycleDeletionProposed:
			pr.Spec.Lifecycle = v1alpha1.PackageRevisionLifecyclePublished
			if err := r.porchClient.Update(r.ctx, pr); err != nil {
				messages = append(messages, err.Error())
				fmt.Fprintf(r.Command.ErrOrStderr(), "%s failed (%s)\n", name, err)
			} else {
				fmt.Fprintf(r.Command.OutOrStderr(), "%s no longer proposed for deletion\n", name)
			}
		default:
			msg := fmt.Sprintf("cannot reject %s with lifecycle '%s'", name, pr.Spec.Lifecycle)
			messages = append(messages, msg)
			fmt.Fprintln(r.Command.ErrOrStderr(), msg)
		}
	}

	if len(messages) > 0 {
		return errors.E(op, fmt.Errorf("errors:\n  %s", strings.Join(messages, "\n  ")))
	}

	return nil
}
