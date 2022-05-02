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

	unversionedapi "github.com/GoogleContainerTools/kpt/porch/api/porch"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/engine"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
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
	// scheme holds our scheme, for type conversions etc
	scheme *runtime.Scheme

	cad engine.CaDEngine
	// coreClient is a client back to the core kubernetes API server, useful for querying CRDs etc
	coreClient     client.Client
	gr             schema.GroupResource
	updateStrategy SimpleRESTUpdateStrategy
}

func (r *packageCommon) listPackages(ctx context.Context, filter packageFilter, callback func(p repository.PackageRevision) error) error {
	var opts []client.ListOption
	if ns, namespaced := genericapirequest.NamespaceFrom(ctx); namespaced {
		opts = append(opts, client.InNamespace(ns))

		if filter.Namespace != "" && ns != filter.Namespace {
			return fmt.Errorf("conflicting namespaces specified: %q and %q", ns, filter.Namespace)
		}
	}

	// TODO: Filter on filter.Repository?
	var repositories configapi.RepositoryList
	if err := r.coreClient.List(ctx, &repositories, opts...); err != nil {
		return fmt.Errorf("error listing repository objects: %w", err)
	}

	for i := range repositories.Items {
		repositoryObj := &repositories.Items[i]

		if filter.Repository != "" && filter.Repository != repositoryObj.GetName() {
			continue
		}

		repository, err := r.cad.OpenRepository(ctx, repositoryObj)
		if err != nil {
			return err
		}

		revisions, err := repository.ListPackageRevisions(ctx, filter.ListPackageRevisionFilter)
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

	repositoryName, err := ParseRepositoryName(name)
	if err != nil {
		return nil, apierrors.NewNotFound(r.gr, name)
	}

	var repositoryObj configapi.Repository
	repositoryID := types.NamespacedName{Namespace: ns, Name: repositoryName}
	if err := r.coreClient.Get(ctx, repositoryID, &repositoryObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apierrors.NewNotFound(r.gr, name)
		}
		return nil, fmt.Errorf("error getting repository %v: %w", repositoryID, err)
	}

	repo, err := r.cad.OpenRepository(ctx, &repositoryObj)
	if err != nil {
		return nil, err
	}

	revisions, err := repo.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{KubeObjectName: name})
	if err != nil {
		return nil, err
	}
	for _, rev := range revisions {
		if rev.KubeObjectName() == name {
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

	obj := pkg.GetPackageRevision()
	return obj, nil
}

// Common implementation of PackageRevision update logic.
func (r *packageCommon) updatePackageRevision(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	// TODO: Is this all boilerplate??

	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, false, apierrors.NewBadRequest("namespace must be specified")
	}

	// isCreate tracks whether this is an update that creates an object (this happens in server-side apply)
	isCreate := false

	oldPackage, err := r.getPackage(ctx, name)
	if err != nil {
		if forceAllowCreate && apierrors.IsNotFound(err) {
			// For server-side apply, we can create the object here
			isCreate = true
		} else {
			return nil, false, err
		}
	}

	var oldRuntimeObj runtime.Object // We have to be runtime.Object (and not *api.PackageRevision) or else nil-checks fail (because a nil object is not a nil interface)
	if !isCreate {
		oldRuntimeObj = oldPackage.GetPackageRevision()
	}

	newRuntimeObj, err := objInfo.UpdatedObject(ctx, oldRuntimeObj)
	if err != nil {
		klog.Infof("update failed to construct UpdatedObject: %v", err)
		return nil, false, err
	}

	// This type conversion is necessary because mutations work with unversioned types
	// (mostly for historical reasons).  So the server-side-apply library returns an unversioned object.
	if unversioned, isUnversioned := newRuntimeObj.(*unversionedapi.PackageRevision); isUnversioned {
		klog.Warningf("converting from unversioned to versioned object")
		typed := &api.PackageRevision{}
		if err := r.scheme.Convert(unversioned, typed, nil); err != nil {
			return nil, false, fmt.Errorf("failed to convert %T to %T: %w", unversioned, typed, err)
		}
		newRuntimeObj = typed
	}

	r.updateStrategy.PrepareForUpdate(ctx, newRuntimeObj, oldRuntimeObj)

	if !isCreate {
		if updateValidation != nil {
			err := updateValidation(ctx, newRuntimeObj, oldRuntimeObj)
			if err != nil {
				klog.Infof("update failed validation: %v", err)
				return nil, false, err
			}
		}

		fieldErrors := r.updateStrategy.ValidateUpdate(ctx, newRuntimeObj, oldRuntimeObj)
		if len(fieldErrors) > 0 {
			return nil, false, apierrors.NewInvalid(api.SchemeGroupVersion.WithKind("PackageRevision").GroupKind(), name, fieldErrors)
		}
	}

	if isCreate {
		if createValidation != nil {
			err := createValidation(ctx, newRuntimeObj)
			if err != nil {
				klog.Infof("update failed create validation: %v", err)
				return nil, false, err
			}
		}

		// TODO: ValidateCreate function ?
		// fieldErrors := r.updateStrategy.ValidateCreate(ctx, newRuntimeObj, oldRuntimeObj)
		// if len(fieldErrors) > 0 {
		// 	return nil, false, apierrors.NewInvalid(api.SchemeGroupVersion.WithKind("PackageRevision").GroupKind(), name, fieldErrors)
		// }
	}

	r.updateStrategy.Canonicalize(newRuntimeObj)

	newObj, ok := newRuntimeObj.(*api.PackageRevision)
	if !ok {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("expected PackageRevision object, got %T", newRuntimeObj))
	}

	repositoryName, err := ParseRepositoryName(name)
	if err != nil {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("invalid name %q", name))
	}
	if isCreate {
		repositoryName = newObj.Spec.RepositoryName
		if repositoryName == "" {
			return nil, false, apierrors.NewBadRequest(fmt.Sprintf("invalid repositoryName %q", name))
		}
	}

	var repositoryObj configapi.Repository
	repositoryID := types.NamespacedName{Namespace: ns, Name: repositoryName}
	if err := r.coreClient.Get(ctx, repositoryID, &repositoryObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, false, apierrors.NewNotFound(configapi.KindRepository.GroupResource(), repositoryID.Name)
		}
		return nil, false, apierrors.NewInternalError(fmt.Errorf("error getting repository %v: %w", repositoryID, err))
	}

	if !isCreate {
		rev, err := r.cad.UpdatePackageRevision(ctx, &repositoryObj, oldPackage, oldRuntimeObj.(*api.PackageRevision), newObj)
		if err != nil {
			return nil, false, apierrors.NewInternalError(err)
		}

		updated := rev.GetPackageRevision()

		return updated, false, nil
	} else {
		rev, err := r.cad.CreatePackageRevision(ctx, &repositoryObj, newObj)
		if err != nil {
			klog.Infof("error creating package: %v", err)
			return nil, false, apierrors.NewInternalError(err)
		}

		created := rev.GetPackageRevision()
		return created, true, nil
	}
}
