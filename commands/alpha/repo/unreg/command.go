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

package unreg

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/repodocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/spf13/cobra"
	coreapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrepounreg"
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
		Use:     "unreg REPOSITORY [flags]",
		Aliases: []string{"unregister"},
		Short:   repodocs.UnregShort,
		Long:    repodocs.UnregShort + "\n" + repodocs.UnregLong,
		Example: repodocs.UnregExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	c.Flags().BoolVar(&r.keepSecret, "keep-auth-secret", false, "Keep the auth secret associated with the repository registration, if any")

	return r
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command

	// Flags
	keepSecret bool
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

	if len(args) == 0 {
		return errors.E(op, fmt.Errorf("REPOSITORY is a required positional argument"))
	}

	repository := args[0]

	var repo configapi.Repository
	if err := r.client.Get(r.ctx, client.ObjectKey{
		Namespace: *r.cfg.Namespace,
		Name:      repository,
	}, &repo); err != nil {
		return errors.E(op, err)
	}
	if err := r.client.Delete(r.ctx, &configapi.Repository{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Repository",
			APIVersion: configapi.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      repo.Name,
			Namespace: repo.Namespace,
		},
	}); err != nil {
		return errors.E(op, err)
	}

	if r.keepSecret {
		return nil
	}

	secret := getSecretName(&repo)
	if secret == "" {
		return nil
	}

	if err := r.client.Delete(r.ctx, &coreapi.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: coreapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret,
			Namespace: repo.Namespace,
		},
	}); err != nil {
		return errors.E(op, fmt.Errorf("failed to delete Secret %s: %w", secret, err))
	}

	return nil
}

func getSecretName(repo *configapi.Repository) string {
	if repo.Spec.Git != nil {
		return repo.Spec.Git.SecretRef.Name
	}
	if repo.Spec.Oci != nil {
		return repo.Spec.Oci.SecretRef.Name
	}
	return ""
}
