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

package cmdregister

import (
	"context"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/parse"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	coreapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	aggregatorv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	c := &cobra.Command{
		Use:        "register REPOSITORY",
		Aliases:    []string{},
		SuggestFor: []string{},
		Short:      "TODO",
		Long:       "TODO",
		Example:    "TODO",
		PreRunE:    r.preRunE,
		RunE:       r.runE,
		Hidden:     true,
	}
	r.Command = c

	c.Flags().StringVar(&r.title, "title", "", "Title of the package repository.")
	c.Flags().StringVar(&r.name, "name", "", "Name of the package repository. If unspecified, will use the name portion (last segment) of the repository URL.")
	c.Flags().StringVar(&r.description, "description", "", "Brief description of the package repository.")

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

	// Flags
	title       string
	name        string
	description string
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = "cmdregister.preRunE"
	config, err := r.cfg.ToRESTConfig()
	if err != nil {
		return errors.E(op, err)
	}

	scheme, err := createScheme()
	if err != nil {
		return errors.E(op, err)
	}

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return errors.E(op, err)
	}

	r.client = c
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = "cmdregister.runE"

	if len(args) == 0 {
		return errors.E(op, "repository is required positional argument")
	}

	repository := args[0]

	var git *configapi.GitRepository
	var oci *configapi.OciRepository
	var rt configapi.RepositoryType

	if strings.HasPrefix(repository, "oci://") {
		rt = configapi.RepositoryTypeOCI
		oci = &configapi.OciRepository{
			Registry: repository[6:],
		}
		if r.name == "" {
			r.name = lastSegment(repository)
		}
	} else {
		rt = configapi.RepositoryTypeGit
		// TODO: better parsing.
		t, err := parse.GitParseArgs(r.ctx, []string{repository, "."})
		if err != nil {
			return errors.E(op, err)
		}
		git = &configapi.GitRepository{
			Repo:      t.Repo,
			Branch:    t.Ref,
			Directory: t.Directory,
			// TODO: support private repositories; accept username, password, create secret
		}

		if r.name == "" {
			r.name = lastSegment(t.Repo)
		}
	}

	if err := r.client.Create(r.ctx, &configapi.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.name,
			Namespace: *r.cfg.Namespace,
		},
		Spec: configapi.RepositorySpec{
			Title:       r.title,
			Description: r.description,
			Type:        rt,
			Content:     configapi.RepositoryContentPackage,
			Git:         git,
			Oci:         oci,
		},
	}); err != nil {
		return errors.E(op, err)
	}

	return nil
}

func createScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	for _, api := range (runtime.SchemeBuilder{
		porchapi.AddToScheme,
		configapi.AddToScheme,
		coreapi.AddToScheme,
		aggregatorv1.AddToScheme,
		appsv1.AddToScheme,
	}) {
		if err := api(scheme); err != nil {
			return nil, err
		}
	}
	return scheme, nil
}

func lastSegment(path string) string {
	path = strings.TrimRight(path, "/")
	return path[strings.LastIndex(path, "/")+1:]
}
