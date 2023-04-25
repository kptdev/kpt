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

package clone

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/commands/alpha/rpkg/util"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/rpkgdocs"
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
		Short:   rpkgdocs.CloneShort,
		Long:    rpkgdocs.CloneShort + "\n" + rpkgdocs.CloneLong,
		Example: rpkgdocs.CloneExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	c.Flags().StringVar(&r.strategy, "strategy", string(porchapi.ResourceMerge),
		"update strategy that should be used when updating this package; one of: "+strings.Join(strategies, ","))
	c.Flags().StringVar(&r.directory, "directory", "", "Directory within the repository where the upstream package is located.")
	c.Flags().StringVar(&r.ref, "ref", "", "Branch in the repository where the upstream package is located.")
	c.Flags().StringVar(&r.repository, "repository", "", "Repository to which package will be cloned (downstream repository).")
	c.Flags().StringVar(&r.workspace, "workspace", "v1", "Workspace name of the downstream package.")

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
	directory  string
	ref        string
	repository string // Target repository
	workspace  string // Target workspaceName
	target     string // Target package name
}

func (r *runner) preRunE(_ *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"
	client, err := porch.CreateClientWithFlags(r.cfg)
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

	if r.workspace == "" {
		return errors.E(op, fmt.Errorf("--workspace is required to specify downstream workspace name"))
	}

	source := args[0]
	target := args[1]

	pkgExists, err := util.PackageAlreadyExists(r.ctx, r.client, r.repository, target, *r.cfg.Namespace)
	if err != nil {
		return err
	}
	if pkgExists {
		return fmt.Errorf("`clone` cannot create a new revision for package %q that already exists in repo %q; make subsequent revisions using `copy`",
			target, r.repository)
	}

	switch {
	case strings.HasPrefix(source, "oci://"):
		r.clone.Upstream.Type = porchapi.RepositoryTypeOCI
		r.clone.Upstream.Oci = &porchapi.OciPackage{
			Image: source,
		}

	case strings.Contains(source, "/"):
		if parse.HasGitSuffix(source) { // extra parsing required
			repo, dir, ref, err := parse.URL(source)
			if err != nil {
				return err
			}
			// throw error if values set by flags contradict values parsed from SOURCE_PACKAGE
			if r.directory != "" && dir != "" && r.directory != dir {
				return errors.E(op, fmt.Errorf("directory %s specified by --directory contradicts directory %s specified by SOURCE_PACKAGE",
					r.directory, dir))
			}
			if r.ref != "" && ref != "" && r.ref != ref {
				return errors.E(op, fmt.Errorf("ref %s specified by --ref contradicts ref %s specified by SOURCE_PACKAGE",
					r.ref, ref))
			}
			// grab the values parsed from SOURCE_PACKAGE
			if r.directory == "" {
				r.directory = dir
			}
			if r.ref == "" {
				r.ref = ref
			}
			source = repo + ".git" // parse.ParseURL removes the git suffix, we need to add it back
		}
		if r.ref == "" {
			r.ref = "main"
		}
		if r.directory == "" {
			r.directory = "/"
		}
		r.clone.Upstream.Type = porchapi.RepositoryTypeGit
		r.clone.Upstream.Git = &porchapi.GitPackage{
			Repo:      source,
			Ref:       r.ref,
			Directory: r.directory,
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

func (r *runner) runE(cmd *cobra.Command, _ []string) error {
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
			WorkspaceName:  porchapi.WorkspaceName(r.workspace),
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

	fmt.Fprintf(cmd.OutOrStdout(), "%s created\n", pr.Name)
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
