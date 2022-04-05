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

package cmdrpkgapprove

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkgapprove"
	longMsg = `
kpt alpha rpkg approve [PACKAGE ...] [flags]

Approves a proposal to finalize a package revision

Args:

PACKAGE:
  Name of the proposed package revisions to approve.
`
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
		Use:     "approve PACKAGE",
		Short:   "Approves a proposal to finalize a package revision.",
		Long:    longMsg,
		Example: "kpt alpha rpkg approve git-repository:package-revision:v3",
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	return r
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  rest.Interface
	Command *cobra.Command

	// Flags
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"

	client, err := porch.CreateRESTClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"
	var messages []string

	namespace := *r.cfg.Namespace

	for _, name := range args {
		switch err := porch.UpdatePackageRevisionApproval(r.ctx, r.client, client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, v1alpha1.PackageRevisionLifecycleFinal); err {
		case nil:
			fmt.Fprintf(r.Command.OutOrStderr(), "%s approved\n", name)
		default:
			messages = append(messages, err.Error())
			fmt.Fprintf(r.Command.ErrOrStderr(), "%s failed (%s)\n", name, err)
		}
	}

	if len(messages) > 0 {
		return errors.E(op, fmt.Errorf("errors:\n  %s", strings.Join(messages, "\n  ")))
	}

	return nil
}
