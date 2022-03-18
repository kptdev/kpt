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
    * repository:package:revision

TARGET:
  Target package name in the format: REPOSITORY[:PACKAGE[:REVISION]]
  Example: package-repository:package-name:v1

Flags:

--strategy
  "update strategy that should be used when updating this package; one of: resource-merge, fast-forward, force-delete-replace
 
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
		Use:     "clone SOURCE_PACKAGE TARGET",
		Short:   "Creates a clone of a source package in the target repository.",
		Long:    longMsg,
		Example: "kpt alpha rpkg clone git-repository:source-package:v2 target-repository:target-package-name:v1",
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	c.Flags().StringVar(&r.strategy, "strategy", string(porchapi.ResourceMerge),
		"update strategy that should be used when updating this package; one of: "+strings.Join(strategies, ","))

	return r
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command

	clone  porchapi.PackageCloneTaskSpec
	target porch.PackageName

	// Flags
	strategy string
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
		return errors.E(op, fmt.Errorf("SOURCE_PACKAGE and TARGET are required positional arguments; %d provided", len(args)))
	}

	source := args[0]
	target := args[1]

	// Source is a KRM package name?
	targetPackageName, nameParts := porch.ParsePartialPackageName(target)
	if nameParts < 1 || nameParts > 3 {
		return errors.E(op, fmt.Errorf("invalid target name: %q", target))
	}

	switch {
	case strings.HasPrefix(source, "oci://"):
		r.clone.Upstream.Type = porchapi.RepositoryTypeOCI
		r.clone.Upstream.Oci = &porchapi.OciPackage{
			Image: source,
		}

		// TODO: Infer target package name from source
		if targetPackageName.Package == "" {
			return errors.E(op, fmt.Errorf("missing target package name (%q)", target))
		}

	case strings.Contains(source, "/"):
		// TODO: better parsing
		git, err := parse.GitParseArgs(r.ctx, []string{source, "."})
		if err != nil {
			return errors.E(op, err)
		}

		r.clone.Upstream.Type = porchapi.RepositoryTypeGit
		r.clone.Upstream.Git = &porchapi.GitPackage{
			Repo: git.Repo,
			Ref:  git.Ref,
			// TODO: Temporary limitation of Porch server - it does not handle leading
			// and trailing '/' in directory names. Can be removed when PR 2913 is merged.
			Directory: strings.Trim(git.Directory, "/"),
		}
		// TODO: support authn
		if targetPackageName.Package == "" {
			targetPackageName.Package = porch.LastSegment(git.Directory)
		}

	default:
		src, err := porch.ParsePackageName(source)
		if err != nil {
			return errors.E(op, err)
		}
		if targetPackageName.Package == "" {
			targetPackageName.Package = src.Package
		}
		r.clone.Upstream.UpstreamRef = porchapi.PackageRevisionRef{
			Name: src.Original,
		}
	}

	if targetPackageName.Revision == "" {
		targetPackageName.Revision = "v1"
	}
	r.target = targetPackageName

	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	if err := r.client.Create(r.ctx, &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.target.Identifier(),
			Namespace: *r.cfg.Namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    r.target.Package,
			Revision:       r.target.Revision,
			RepositoryName: r.target.Repository,
			Tasks: []porchapi.Task{
				{
					Type:  porchapi.TaskTypeClone,
					Clone: &r.clone,
				},
			},
		},
	}); err != nil {
		return errors.E(op, err)
	}
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
