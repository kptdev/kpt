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

package delete

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/spf13/cobra"
	coreapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdsync.delete"
	longMsg = `
kpt alpha sync delete NAME [flags]

Deletes a package RootSync resource.

Args:

NAME:
  Name of the sync resource. Required argument.

Flags:

--keep-auth-secret
  Do not delete the repository authentication secret, if it exists.
`
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
		Use:     "del REPOSITORY [flags]",
		Aliases: []string{"delete"},
		Short:   "Deletes the package RootSync.",
		Long:    longMsg,
		Example: "kpt alpha sync del deployed-blueprint",
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	c.Flags().BoolVar(&r.keepSecret, "keep-auth-secret", false, "Keep the auth secret associated with the RootSync resource, if any")

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

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"
	client, err := porch.CreateDynamicClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	if len(args) == 0 {
		return errors.E(op, fmt.Errorf("NAME is a required positional argument"))
	}

	name := args[0]
	namespace := *r.cfg.Namespace
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	rs := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "configsync.gke.io/v1beta1",
			"kind":       "RootSync",
		},
	}
	if err := r.client.Get(r.ctx, key, &rs); err != nil {
		return errors.E(op, fmt.Errorf("cannot get %s: %v", key, err))
	}

	if err := r.client.Delete(r.ctx, &rs); err != nil {
		return errors.E(op, err)
	}

	if r.keepSecret {
		return nil
	}

	secret := getSecretName(&rs)
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
			Namespace: namespace,
		},
	}); err != nil {
		return errors.E(op, fmt.Errorf("failed to delete Secret %s: %w", secret, err))
	}

	return nil
}

func getSecretName(repo *unstructured.Unstructured) string {
	name, _, _ := unstructured.NestedString(repo.Object, "spec", "git", "secretRef", "name")
	return name
}
