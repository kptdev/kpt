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

package cmdrpkgupdate

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkgupdate"
)

func NewCommand(ctx context.Context, rcg *genericclioptions.ConfigFlags) *cobra.Command {
	return newRunner(ctx, rcg).Command
}

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	r.Command = &cobra.Command{
		Use:     "update SOURCE_PACKAGE",
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command.Flags().StringVar(&r.revision, "revision", "", "Revision of the upstream package to update to.")
	r.Command.Flags().BoolVar(&r.discover, "discover", false, "If set, search for available updates instead of performing an update.")
	return r
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command

	revision string // Target package revision
	discover bool   // If set, discover updates rather than do updates

	// for --discover, there are multiple places where we need access to all package revisions, so
	// we store it in the runner
	prs []porchapi.PackageRevision
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"
	c, err := porch.CreateClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = c

	if !r.discover {
		if len(args) < 1 {
			return errors.E(op, fmt.Errorf("SOURCE_PACKAGE is a required positional argument"))
		}
		if len(args) > 1 {
			return errors.E(op, fmt.Errorf("too many arguments; SOURCE_PACKAGE is the only accepted positional arguments"))
		}
		// TODO: This should use the latest available revision if one isn't specified.
		if r.revision == "" {
			return errors.E(op, fmt.Errorf("revision is required"))
		}
	}

	packageRevisionList := porchapi.PackageRevisionList{}
	if err := r.client.List(r.ctx, &packageRevisionList, &client.ListOptions{}); err != nil {
		return errors.E(op, err)
	}
	r.prs = packageRevisionList.Items

	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	if r.discover {
		return r.discoverUpdates(cmd, args)
	}

	pr := r.findPackageRevision(args[0])

	cloneTask, found := r.findCloneTask(pr)
	if !found {
		err := fmt.Errorf("upstream source not found. Only cloned packages can be updated")
		return errors.E(op, err)
	}

	switch cloneTask.Clone.Upstream.Type {
	case porchapi.RepositoryTypeGit:
		cloneTask.Clone.Upstream.Git.Ref = r.revision
	case porchapi.RepositoryTypeOCI:
		err := fmt.Errorf("update not implemented for oci packages")
		return errors.E(op, err)
	default:
		upstreamPr := r.findPackageRevision(cloneTask.Clone.Upstream.UpstreamRef.Name)
		if upstreamPr == nil {
			err := fmt.Errorf("upstream package revision %s no longer exists", cloneTask.Clone.Upstream.UpstreamRef.Name)
			return errors.E(op, err)
		}
		newUpstreamPr := r.findPackageRevisionForRef(upstreamPr.Spec.PackageName)
		if newUpstreamPr == nil {
			err := fmt.Errorf("revision %s does not exist for package %s", r.revision, pr.Spec.PackageName)
			return errors.E(op, err)
		}
		cloneTask.Clone.Upstream.UpstreamRef.Name = newUpstreamPr.Name
	}

	if err := r.client.Update(r.ctx, pr); err != nil {
		return errors.E(op, err)
	}
	return nil
}

func (r *runner) findCloneTask(pr *porchapi.PackageRevision) (*porchapi.Task, bool) {
	for i := len(pr.Spec.Tasks) - 1; i >= 0; i-- {
		t := pr.Spec.Tasks[i]
		if t.Type == porchapi.TaskTypeClone {
			return &t, true
		}
	}
	return nil, false
}

func (r *runner) findPackageRevision(prName string) *porchapi.PackageRevision {
	for i := range r.prs {
		pr := r.prs[i]
		if pr.Name == prName {
			return &pr
		}
	}
	return nil
}

func (r *runner) findPackageRevisionForRef(name string) *porchapi.PackageRevision {
	for i := range r.prs {
		pr := r.prs[i]
		if pr.Spec.PackageName == name && pr.Spec.Revision == r.revision {
			return &pr
		}
	}
	return nil
}
