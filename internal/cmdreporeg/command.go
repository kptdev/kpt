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

package cmdreporeg

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/parse"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/spf13/cobra"
	coreapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdreporeg"
	longMsg = `
kpt alpha repo reg[ister] REPOSITORY [flags]

Args:

REPOSITORY:
	Address of the repository to register. Required argument.

Flags:

--description
	Brief description of the package repository.

--name
	Name of the package repository. If unspecified, will use the name portion (last segment) of the repository URL.

--title
	Title of the package repository.

--deployment
  Repository is a deployment repository; packages in a deployment repository are considered deployment-ready.

--repo-username
	Username for repository authentication.

--repo-password
	Password for repository authentication.
`
)

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	c := &cobra.Command{
		Hidden:  true,
		Use:     "reg REPOSITORY",
		Aliases: []string{"register"},
		Short:   "Registers a package repository with Package Orchestrator.",
		Long:    longMsg,
		Example: "TODO",
		PreRunE: r.preRunE,
		RunE:    r.runE,
	}
	r.Command = c

	c.Flags().StringVar(&r.title, "title", "", "Title of the package repository.")
	c.Flags().StringVar(&r.name, "name", "", "Name of the package repository. If unspecified, will use the name portion (last segment) of the repository URL.")
	c.Flags().StringVar(&r.description, "description", "", "Brief description of the package repository.")
	c.Flags().BoolVar(&r.deployment, "deployment", false, "Repository is a deployment repository; packages in a deployment repository are considered deployment-ready.")
	c.Flags().StringVar(&r.username, "repo-username", "", "Username for repository authentication.")
	c.Flags().StringVar(&r.password, "repo-password", "", "Password for repository authentication.")

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
	deployment  bool
	username    string
	password    string
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
		}

		if r.name == "" {
			r.name = lastSegment(t.Repo)
		}
	}

	if r.username != "" || r.password != "" {
		secretName := fmt.Sprintf("%s-auth", r.name)
		if err := apply(r.ctx, r.client, &coreapi.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: coreapi.SchemeGroupVersion.Identifier(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: *r.cfg.Namespace,
			},
			Data: map[string][]byte{
				"username": []byte(r.username),
				"password": []byte(r.password),
			},
			Type: coreapi.SecretTypeBasicAuth,
		}); err != nil {
			return errors.E(op, err)
		}

		if git != nil {
			git.SecretRef.Name = secretName
		}
		if oci != nil {
			oci.SecretRef.Name = secretName
		}
	}

	if err := apply(r.ctx, r.client, &configapi.Repository{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Repository",
			APIVersion: configapi.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.name,
			Namespace: *r.cfg.Namespace,
		},
		Spec: configapi.RepositorySpec{
			Title:       r.title,
			Description: r.description,
			Type:        rt,
			Content:     configapi.RepositoryContentPackage,
			Deployment:  r.deployment,
			Git:         git,
			Oci:         oci,
		},
	}); err != nil {
		return errors.E(op, err)
	}

	return nil
}

func lastSegment(path string) string {
	path = strings.TrimRight(path, "/")
	return path[strings.LastIndex(path, "/")+1:]
}

func apply(ctx context.Context, api client.Client, obj client.Object) error {
	//	api.Create(ctx, obj)
	return api.Patch(ctx, obj, client.Apply, client.FieldOwner("kubectl"))
}
