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

package porch

import (
	"context"
	"fmt"

	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/engine"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type packageCommon struct {
	cad engine.CaDEngine
	// coreClient is a client back to the core kubernetes API server, useful for querying CRDs etc
	coreClient client.Client

	gr schema.GroupResource
}

func (r *packageCommon) listPackages(ctx context.Context, callback func(p repository.PackageRevision) error) error {
	var opts []client.ListOption
	if ns, namespaced := genericapirequest.NamespaceFrom(ctx); namespaced {
		opts = append(opts, client.InNamespace(ns))
	}

	var repositories configapi.RepositoryList
	if err := r.coreClient.List(ctx, &repositories, opts...); err != nil {
		return fmt.Errorf("error listing repository objects: %w", err)
	}

	for i := range repositories.Items {
		repositoryObj := &repositories.Items[i]

		repository, err := r.cad.OpenRepository(repositoryObj)
		if err != nil {
			return err
		}

		revisions, err := repository.ListPackageRevisions(ctx)
		if err != nil {
			return err
		}
		for _, rev := range revisions {
			if err := callback(rev); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *packageCommon) getPackage(ctx context.Context, name string) (repository.PackageRevision, error) {
	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, fmt.Errorf("namespace must be specified")
	}

	nameTokens, err := ParseName(name)
	if err != nil {
		return nil, apierrors.NewNotFound(r.gr, name)
	}

	var repositoryObj configapi.Repository
	repositoryID := types.NamespacedName{Namespace: ns, Name: nameTokens.RepositoryName}
	if err := r.coreClient.Get(ctx, repositoryID, &repositoryObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apierrors.NewNotFound(r.gr, name)
		}
		return nil, fmt.Errorf("error getting repository %v: %w", repositoryID, err)
	}

	repository, err := r.cad.OpenRepository(&repositoryObj)
	if err != nil {
		return nil, err
	}

	revisions, err := repository.ListPackageRevisions(ctx)
	if err != nil {
		return nil, err
	}
	for _, rev := range revisions {
		if rev.Name() == name {
			return rev, nil
		}
	}

	return nil, apierrors.NewNotFound(r.gr, name)
}
