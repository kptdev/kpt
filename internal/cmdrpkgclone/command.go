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

package cmdrpkgclone

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/parse"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkgclone"
	longMsg = `
kpt alpha rpkg clone SOURCE_PACKAGE TARGET

Creates a clone of a source package in the target repository.

Args:

SOURCE_PACKAGE:
  Source package. Can be a reference to an OCI package, Git package, or an package resource name:
    * oci://oci-repository/package-name
    * http://git-repository.git/package-name
    * package-revision-name

NAME:
  Target package revision name (downstream package)
  Example: package-name

Flags:

--repository
  Repository to which package will be cloned (downstream repository).

--revision
  Revision of the downstream package.

--strategy
  Update strategy that should be used when updating this package; one of: resource-merge, fast-forward, force-delete-replace
`
)

var (
	strategies = []string{
		string(porchapi.ResourceMerge),
		string(porchapi.FastForward),
		string(porchapi.ForceDeleteReplace),
	}
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
		Use:     "clone SOURCE_PACKAGE NAME",
		Short:   "Creates a clone of a source package in the target repository.",
		Long:    longMsg,
		Example: "kpt alpha rpkg clone upstream-package-name target-package-name --repository target-repository --revision v1",
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	c.Flags().StringVar(&r.strategy, "strategy", string(porchapi.ResourceMerge),
		"update strategy that should be used when updating this package; one of: "+strings.Join(strategies, ","))
	c.Flags().StringVar(&r.repository, "repository", "", "Repository to which package will be cloned (downstream repository).")
	c.Flags().StringVar(&r.revision, "revision", "v1", "Revision of the downstream package.")

	return r
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command

	clone porchapi.PackageCloneTaskSpec

	// Flags
	strategy   string
	repository string // Target repository
	revision   string // Target package revision
	target     string // Target package name
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"
	client, err := porch.CreateClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client

	mergeStrategy, err := toMergeStrategy(r.strategy)
	if err != nil {
		return errors.E(op, err)
	}
	r.clone.Strategy = mergeStrategy

	if len(args) < 2 {
		return errors.E(op, fmt.Errorf("SOURCE_PACKAGE and NAME are required positional arguments; %d provided", len(args)))
	}

	if r.repository == "" {
		return errors.E(op, fmt.Errorf("--repository is required to specify downstream repository"))
	}

	source := args[0]
	target := args[1]

	switch {
	case strings.HasPrefix(source, "oci://"):
		r.clone.Upstream.Type = porchapi.RepositoryTypeOCI
		r.clone.Upstream.Oci = &porchapi.OciPackage{
			Image: source,
		}

	case strings.Contains(source, "/"):
		gitArgs, err := parse.GitParseArgs(context.Background(), args)
		if err != nil {
			return err
		}
		r.clone.Upstream.Type = porchapi.RepositoryTypeGit
		r.clone.Upstream.Git = &porchapi.GitPackage{
			Repo:      gitArgs.Repo,
			Ref:       gitArgs.Ref,
			Directory: gitArgs.Directory,
		}
		// TODO: support authn

	default:
		r.clone.Upstream.UpstreamRef = &porchapi.PackageRevisionRef{
			Name: source,
		}
	}

	r.target = target

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
			PackageName:    r.target,
			Revision:       r.revision,
			RepositoryName: r.repository,
			Tasks: []porchapi.Task{
				{
					Type:  porchapi.TaskTypeClone,
					Clone: &r.clone,
				},
			},
		},
	}
	if err := r.client.Create(r.ctx, pr); err != nil {
		return errors.E(op, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s created", pr.Name)
	return nil
}

func toMergeStrategy(strategy string) (porchapi.PackageMergeStrategy, error) {
	switch strategy {
	case string(porchapi.ResourceMerge):
		return porchapi.ResourceMerge, nil
	case string(porchapi.FastForward):
		return porchapi.FastForward, nil
	case string(porchapi.ForceDeleteReplace):
		return porchapi.ForceDeleteReplace, nil
	default:
		return "", fmt.Errorf("invalid strategy: %q", strategy)
	}
}
