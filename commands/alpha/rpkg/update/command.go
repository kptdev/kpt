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

package update

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/rpkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkgupdate"

	upstream   = "upstream"
	downstream = "downstream"
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
		Short:   rpkgdocs.UpdateShort,
		Long:    rpkgdocs.UpdateShort + "\n" + rpkgdocs.UpdateLong,
		Example: rpkgdocs.UpdateExamples,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command.Flags().StringVar(&r.revision, "revision", "", "Revision of the upstream package to update to.")
	r.Command.Flags().StringVar(&r.discover, "discover", "",
		`If set, search for available updates instead of performing an update. 
Setting this to 'upstream' will discover upstream updates of downstream packages.
Setting this to 'downstream' will discover downstream package revisions of upstream packages that need to be updated.`)
	return r
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command

	revision string // Target package revision
	discover string // If set, discover updates rather than do updates

	// there are multiple places where we need access to all package revisions, so
	// we store it in the runner
	prs []porchapi.PackageRevision
}

func (r *runner) preRunE(_ *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"
	c, err := porch.CreateClientWithFlags(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = c

	if r.discover == "" {
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
	} else if r.discover != upstream && r.discover != downstream {
		return errors.E(op, fmt.Errorf("argument for 'discover' must be one of 'upstream' or 'downstream'"))
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

	if r.discover == "" {
		pr := r.findPackageRevision(args[0])
		if pr == nil {
			return errors.E(op, fmt.Errorf("could not find package revision %s", args[0]))
		}
		if err := r.doUpdate(pr); err != nil {
			return errors.E(op, err)
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s updated\n", pr.Name); err != nil {
			return errors.E(op, err)
		}
	} else if err := r.discoverUpdates(cmd, args); err != nil {
		return errors.E(op, err)
	}
	return nil
}

func (r *runner) doUpdate(pr *porchapi.PackageRevision) error {
	cloneTask := r.findCloneTask(pr)
	if cloneTask == nil {
		return fmt.Errorf("upstream source not found for package rev %q; only cloned packages can be updated", pr.Spec.PackageName)
	}

	switch cloneTask.Clone.Upstream.Type {
	case porchapi.RepositoryTypeGit:
		cloneTask.Clone.Upstream.Git.Ref = r.revision
	case porchapi.RepositoryTypeOCI:
		return fmt.Errorf("update not implemented for oci packages")
	default:
		upstreamPr := r.findPackageRevision(cloneTask.Clone.Upstream.UpstreamRef.Name)
		if upstreamPr == nil {
			return fmt.Errorf("upstream package revision %s no longer exists", cloneTask.Clone.Upstream.UpstreamRef.Name)
		}
		newUpstreamPr := r.findPackageRevisionForRef(upstreamPr.Spec.PackageName)
		if newUpstreamPr == nil {
			return fmt.Errorf("revision %s does not exist for package %s", r.revision, pr.Spec.PackageName)
		}
		newTask := porchapi.Task{
			Type: porchapi.TaskTypeUpdate,
			Update: &porchapi.PackageUpdateTaskSpec{
				Upstream: cloneTask.Clone.Upstream,
			},
		}
		newTask.Update.Upstream.UpstreamRef.Name = newUpstreamPr.Name
		pr.Spec.Tasks = append(pr.Spec.Tasks, newTask)
	}

	return r.client.Update(r.ctx, pr)
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

func (r *runner) findCloneTask(pr *porchapi.PackageRevision) *porchapi.Task {
	if len(pr.Spec.Tasks) == 0 {
		return nil
	}
	firstTask := pr.Spec.Tasks[0]
	if firstTask.Type == porchapi.TaskTypeClone {
		return &firstTask
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
