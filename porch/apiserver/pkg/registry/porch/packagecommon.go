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

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/engine"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type packageCommon struct {
	cad engine.CaDEngine
	// coreClient is a client back to the core kubernetes API server, useful for querying CRDs etc
	coreClient     client.Client
	gr             schema.GroupResource
	updateStrategy SimpleRESTUpdateStrategy
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

		repository, err := r.cad.OpenRepository(ctx, repositoryObj)
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

	repository, err := r.cad.OpenRepository(ctx, &repositoryObj)
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

func (r *packageCommon) getPackageRevision(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	pkg, err := r.getPackage(ctx, name)
	if err != nil {
		return nil, err
	}

	obj, err := pkg.GetPackageRevision()
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Common implementation of PackageRevision update logic.
func (r *packageCommon) updatePackageRevision(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	// TODO: Is this all boilerplate??

	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, false, apierrors.NewBadRequest("namespace must be specified")
	}

	oldPackage, err := r.getPackage(ctx, name)
	if err != nil {
		return nil, false, err
	}

	oldObj, err := oldPackage.GetPackageRevision()
	if err != nil {
		klog.Infof("update failed to retrieve old object: %v", err)
		return nil, false, err
	}

	newRuntimeObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		klog.Infof("update failed to construct UpdatedObject: %v", err)
		return nil, false, err
	}

	r.updateStrategy.PrepareForUpdate(ctx, newRuntimeObj, oldObj)

	if updateValidation != nil {
		err := updateValidation(ctx, newRuntimeObj, oldObj)
		if err != nil {
			klog.Infof("update failed validation: %v", err)
			return nil, false, err
		}
	}

	fieldErrors := r.updateStrategy.ValidateUpdate(ctx, newRuntimeObj, oldObj)
	if len(fieldErrors) > 0 {
		return nil, false, apierrors.NewInvalid(api.SchemeGroupVersion.WithKind("PackageRevision").GroupKind(), oldObj.Name, fieldErrors)
	}
	r.updateStrategy.Canonicalize(newRuntimeObj)

	newObj, ok := newRuntimeObj.(*api.PackageRevision)
	if !ok {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("expected PackageRevision object, got %T", newRuntimeObj))
	}

	nameTokens, err := ParseName(name)
	if err != nil {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("invalid name %q", name))
	}

	var repositoryObj configapi.Repository
	repositoryID := types.NamespacedName{Namespace: ns, Name: nameTokens.RepositoryName}
	if err := r.coreClient.Get(ctx, repositoryID, &repositoryObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, false, apierrors.NewNotFound(api.PackageRevisionResourcesGVR.GroupResource(), repositoryID.Name)
		}
		return nil, false, apierrors.NewInternalError(fmt.Errorf("error getting repository %v: %w", repositoryID, err))
	}

	rev, err := r.cad.UpdatePackageRevision(ctx, &repositoryObj, oldPackage, oldObj, newObj)
	if err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}

	created, err := rev.GetPackageRevision()
	if err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}
	return created, false, nil
}
