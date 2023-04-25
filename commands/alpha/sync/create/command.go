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

package create

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/commands/util"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/syncdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/spf13/cobra"
	coreapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdsync.create"
)

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx: ctx,
		cfg: rcg,
	}
	c := &cobra.Command{
		Use:     "create NAME",
		Short:   syncdocs.CreateShort,
		Long:    syncdocs.CreateShort + "\n" + syncdocs.CreateLong,
		Example: syncdocs.CreateExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	c.Flags().StringVar(&r.syncPkg, "package", "", "Name of the package revision to sync. Required.")

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
	syncPkg string
}

func (r *runner) preRunE(_ *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"

	if len(args) == 0 {
		return errors.E(op, "NAME is required positional argument")
	}
	if r.syncPkg == "" {
		return errors.E(op, "--package is a required flag")
	}

	client, err := porch.CreateDynamicClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}

	r.client = client
	return nil
}

func (r *runner) runE(_ *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	syncName := args[0]

	var pr porchapi.PackageRevision
	if err := r.client.Get(r.ctx, client.ObjectKey{
		Namespace: *r.cfg.Namespace,
		Name:      r.syncPkg,
	}, &pr); err != nil {
		return errors.E(op, err)
	}

	var repository configapi.Repository
	if err := r.client.Get(r.ctx, client.ObjectKey{
		Namespace: *r.cfg.Namespace,
		Name:      pr.Spec.RepositoryName,
	}, &repository); err != nil {
		return errors.E(op, err)
	}

	if repository.Spec.Type != configapi.RepositoryTypeGit {
		return errors.E(op, fmt.Sprintf("repository %s/%s is not a git repository; %s is not supported",
			repository.Namespace, repository.Name, repository.Spec.Type))
	}
	if repository.Spec.Git == nil {
		return errors.E(op, fmt.Sprintf("repository %s/%s is missing Git spec", repository.Namespace, repository.Name))
	}

	var secret coreapi.Secret

	if secretName := repository.Spec.Git.SecretRef.Name; secretName != "" {
		var repositorySecret coreapi.Secret
		key := client.ObjectKey{Namespace: *r.cfg.Namespace, Name: repository.Spec.Git.SecretRef.Name}
		if err := r.client.Get(r.ctx, key, &repositorySecret); err != nil {
			return errors.E(op, fmt.Sprintf("cannot retrieve repository credentials %s: %v", key, err))
		}

		secret = coreapi.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: coreapi.SchemeGroupVersion.Identifier(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-auth", syncName),
				Namespace: util.RootSyncNamespace,
			},
			Data: map[string][]byte{
				"username": repositorySecret.Data["username"],
				"token":    repositorySecret.Data["password"],
			},
		}

		if err := porch.Apply(r.ctx, r.client, &secret); err != nil {
			return errors.E(op, err)
		}
	}

	git := map[string]interface{}{
		"repo":     repository.Spec.Git.Repo,
		"revision": fmt.Sprintf("%s/%s", pr.Spec.PackageName, pr.Spec.Revision),
		"dir":      pr.Spec.PackageName,
		"branch":   repository.Spec.Git.Branch,
	}

	if secret.Name != "" {
		git["auth"] = "token"
		git["secretRef"] = map[string]interface{}{
			"name": secret.Name,
		}
	}

	rootsync := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "configsync.gke.io/v1beta1",
			"kind":       "RootSync",
			"metadata": map[string]interface{}{
				"name":      syncName,
				"namespace": util.RootSyncNamespace,
			},
			"spec": map[string]interface{}{
				"sourceFormat": "unstructured",
				"git":          git,
			},
		},
	}

	fmt.Println(rootsync.GetName())
	fmt.Println(rootsync.GetNamespace())

	if err := porch.Apply(r.ctx, r.client, rootsync); err != nil {
		return errors.E(op, err)
	}

	fmt.Fprintf(r.Command.OutOrStderr(), "Created RootSync config-management-system/%s", syncName)
	return nil
}
