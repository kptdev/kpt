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

package delete

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleContainerTools/kpt/commands/util"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/syncdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/spf13/cobra"
	coreapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command         = "cmdsync.delete"
	emptyRepo       = "https://github.com/platkrm/empty"
	emptyRepoBranch = "main"
	defaultTimeout  = 2 * time.Minute
)

var (
	rootSyncGVK = schema.GroupVersionKind{
		Group:   "configsync.gke.io",
		Version: "v1beta1",
		Kind:    "RootSync",
	}
	resourceGroupGVK = schema.GroupVersionKind{
		Group:   "kpt.dev",
		Version: "v1alpha1",
		Kind:    "ResourceGroup",
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
		Use:     "del REPOSITORY [flags]",
		Aliases: []string{"delete"},
		Short:   syncdocs.DeleteShort,
		Long:    syncdocs.DeleteShort + "\n" + syncdocs.DeleteLong,
		Example: syncdocs.DeleteExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	c.Flags().BoolVar(&r.keepSecret, "keep-auth-secret", false, "Keep the auth secret associated with the RootSync resource, if any")
	c.Flags().DurationVar(&r.timeout, "timeout", defaultTimeout, "How long to wait for Config Sync to delete package RootSync")

	return r
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.WithWatch
	Command *cobra.Command

	// Flags
	keepSecret bool
	timeout    time.Duration
}

func (r *runner) preRunE(_ *cobra.Command, _ []string) error {
	const op errors.Op = command + ".preRunE"
	client, err := porch.CreateDynamicClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client
	return nil
}

func (r *runner) runE(_ *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	if len(args) == 0 {
		return errors.E(op, fmt.Errorf("NAME is a required positional argument"))
	}

	name := args[0]
	namespace := util.RootSyncNamespace
	if *r.cfg.Namespace != "" {
		namespace = *r.cfg.Namespace
	}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	rs := unstructured.Unstructured{}
	rs.SetGroupVersionKind(rootSyncGVK)
	if err := r.client.Get(r.ctx, key, &rs); err != nil {
		return errors.E(op, fmt.Errorf("cannot get %s: %v", key, err))
	}

	git, found, err := unstructured.NestedMap(rs.Object, "spec", "git")
	if err != nil || !found {
		return errors.E(op, fmt.Errorf("couldn't find `spec.git`: %v", err))
	}

	git["repo"] = emptyRepo
	git["branch"] = emptyRepoBranch
	git["dir"] = ""
	git["revision"] = ""

	if err := unstructured.SetNestedMap(rs.Object, git, "spec", "git"); err != nil {
		return errors.E(op, err)
	}

	fmt.Println("Deleting synced resources..")
	if err := r.client.Update(r.ctx, &rs); err != nil {
		return errors.E(op, err)
	}

	if err := func() error {
		ctx, cancel := context.WithTimeout(r.ctx, r.timeout)
		defer cancel()

		if err := r.waitForRootSync(ctx, name, namespace); err != nil {
			return err
		}

		fmt.Println("Waiting for deleted resources to be removed..")
		return r.waitForResourceGroup(ctx, name, namespace)
	}(); err != nil {
		// TODO: See if we can expose more information here about what might have prevented a package
		// from being deleted.
		e := fmt.Errorf("package %s failed to be deleted after %f seconds: %v", name, r.timeout.Seconds(), err)
		return errors.E(op, e)
	}

	if err := r.client.Delete(r.ctx, &rs); err != nil {
		return errors.E(op, fmt.Errorf("failed to clean up RootSync: %w", err))
	}

	rg := unstructured.Unstructured{}
	rg.SetGroupVersionKind(resourceGroupGVK)
	rg.SetName(rs.GetName())
	rg.SetNamespace(rs.GetNamespace())
	if err := r.client.Delete(r.ctx, &rg); err != nil {
		return errors.E(op, fmt.Errorf("failed to clean up ResourceGroup: %w", err))
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

	fmt.Printf("Sync %s successfully deleted\n", name)
	return nil
}

func (r *runner) waitForRootSync(ctx context.Context, name string, namespace string) error {
	const op errors.Op = command + ".waitForRootSync"

	return r.waitForResource(ctx, resourceGroupGVK, name, namespace, func(u *unstructured.Unstructured) (bool, error) {
		res, err := status.Compute(u)
		if err != nil {
			return false, errors.E(op, err)
		}
		if res.Status == status.CurrentStatus {
			return true, nil
		}
		return false, nil
	})
}

func (r *runner) waitForResourceGroup(ctx context.Context, name string, namespace string) error {
	const op errors.Op = command + ".waitForResourceGroup"

	return r.waitForResource(ctx, resourceGroupGVK, name, namespace, func(u *unstructured.Unstructured) (bool, error) {
		resources, found, err := unstructured.NestedSlice(u.Object, "spec", "resources")
		if err != nil {
			return false, errors.E(op, err)
		}
		if !found {
			return true, nil
		}
		if len(resources) == 0 {
			return true, nil
		}
		return false, nil
	})
}

type ReconcileFunc func(*unstructured.Unstructured) (bool, error)

func (r *runner) waitForResource(ctx context.Context, gvk schema.GroupVersionKind, name, namespace string, reconcileFunc ReconcileFunc) error {
	const op errors.Op = command + ".waitForResource"

	u := unstructured.UnstructuredList{}
	u.SetGroupVersionKind(gvk)
	watch, err := r.client.Watch(r.ctx, &u)
	if err != nil {
		return errors.E(op, err)
	}
	defer watch.Stop()

	for {
		select {
		case ev, ok := <-watch.ResultChan():
			if !ok {
				return errors.E(op, fmt.Errorf("watch closed unexpectedly"))
			}
			if ev.Object == nil {
				continue
			}

			u := ev.Object.(*unstructured.Unstructured)

			if u.GetName() != name || u.GetNamespace() != namespace {
				continue
			}

			reconciled, err := reconcileFunc(u)
			if err != nil {
				return err
			}
			if reconciled {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func getSecretName(repo *unstructured.Unstructured) string {
	name, _, _ := unstructured.NestedString(repo.Object, "spec", "git", "secretRef", "name")
	return name
}
