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
	"k8s.io/apimachinery/pkg/labels"
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
	createStrategy SimpleRESTCreateStrategy
}

func (r *packageCommon) listPackageRevisions(ctx context.Context, filter packageRevisionFilter, selector labels.Selector, callback func(p *engine.PackageRevision) error) error {
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

		revisions, err := r.cad.ListPackageRevisions(ctx, repositoryObj, filter.ListPackageRevisionFilter)
		if err != nil {
			klog.Warningf("error listing package revisions from repository %s/%s: %s", repositoryObj.GetNamespace(), repositoryObj.GetName(), err)
			continue
		}
		for _, rev := range revisions {
			apiPkgRev, err := rev.GetPackageRevision(ctx)
			if err != nil {
				return err
			}

			if selector != nil && !selector.Matches(labels.Set(apiPkgRev.Labels)) {
				continue
			}

			if err := callback(rev); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *packageCommon) listPackages(ctx context.Context, filter packageFilter, callback func(p *engine.Package) error) error {
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

		revisions, err := r.cad.ListPackages(ctx, repositoryObj, filter.ListPackageFilter)
		if err != nil {
			klog.Warningf("error listing packages from repository %s/%s: %s", repositoryObj.GetNamespace(), repositoryObj.GetName(), err)
			continue
		}
		for _, rev := range revisions {
			if err := callback(rev); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *packageCommon) watchPackages(ctx context.Context, filter packageRevisionFilter, callback engine.ObjectWatcher) error {
	if err := r.cad.ObjectCache().WatchPackageRevisions(ctx, filter.ListPackageRevisionFilter, callback); err != nil {
		return err
	}

	return nil
}

func (r *packageCommon) getRepositoryObjFromName(ctx context.Context, name string) (*configapi.Repository, error) {
	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, fmt.Errorf("namespace must be specified")
	}
	repositoryName, err := ParseRepositoryName(name)
	if err != nil {
		return nil, apierrors.NewNotFound(r.gr, name)
	}

	return r.getRepositoryObj(ctx, types.NamespacedName{Name: repositoryName, Namespace: ns})
}

func (r *packageCommon) getRepositoryObj(ctx context.Context, repositoryID types.NamespacedName) (*configapi.Repository, error) {
	var repositoryObj configapi.Repository
	if err := r.coreClient.Get(ctx, repositoryID, &repositoryObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apierrors.NewNotFound(configapi.TypeRepository.GroupResource(), repositoryID.Name)
		}
		return nil, apierrors.NewInternalError(fmt.Errorf("error getting repository %v: %w", repositoryID, err))
	}
	return &repositoryObj, nil
}

func (r *packageCommon) getRepoPkgRev(ctx context.Context, name string) (*engine.PackageRevision, error) {
	repositoryObj, err := r.getRepositoryObjFromName(ctx, name)
	if err != nil {
		return nil, err
	}
	revisions, err := r.cad.ListPackageRevisions(ctx, repositoryObj, repository.ListPackageRevisionFilter{KubeObjectName: name})
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

func (r *packageCommon) getPackage(ctx context.Context, name string) (*engine.Package, error) {
	repositoryObj, err := r.getRepositoryObjFromName(ctx, name)
	if err != nil {
		return nil, err
	}

	revisions, err := r.cad.ListPackages(ctx, repositoryObj, repository.ListPackageFilter{KubeObjectName: name})
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

// Common implementation of PackageRevision update logic.
func (r *packageCommon) updatePackageRevision(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	// TODO: Is this all boilerplate??

	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, false, apierrors.NewBadRequest("namespace must be specified")
	}

	// isCreate tracks whether this is an update that creates an object (this happens in server-side apply)
	isCreate := false

	oldRepoPkgRev, err := r.getRepoPkgRev(ctx, name)
	if err != nil {
		if forceAllowCreate && apierrors.IsNotFound(err) {
			// For server-side apply, we can create the object here
			isCreate = true
		} else {
			return nil, false, err
		}
	}

	var oldApiPkgRev runtime.Object // We have to be runtime.Object (and not *api.PackageRevision) or else nil-checks fail (because a nil object is not a nil interface)
	if !isCreate {
		oldApiPkgRev, err = oldRepoPkgRev.GetPackageRevision(ctx)
		if err != nil {
			return nil, false, err
		}
	}

	newRuntimeObj, err := objInfo.UpdatedObject(ctx, oldApiPkgRev)
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

	if err := r.validateUpdate(ctx, newRuntimeObj, oldApiPkgRev, isCreate, createValidation,
		updateValidation, "PackageRevision", name); err != nil {
		return nil, false, err
	}

	newApiPkgRev, ok := newRuntimeObj.(*api.PackageRevision)
	if !ok {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("expected PackageRevision object, got %T", newRuntimeObj))
	}

	repositoryName, err := ParseRepositoryName(name)
	if err != nil {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("invalid name %q", name))
	}
	if isCreate {
		repositoryName = newApiPkgRev.Spec.RepositoryName
		if repositoryName == "" {
			return nil, false, apierrors.NewBadRequest(fmt.Sprintf("invalid repositoryName %q", name))
		}
	}

	var repositoryObj configapi.Repository
	repositoryID := types.NamespacedName{Namespace: ns, Name: repositoryName}
	if err := r.coreClient.Get(ctx, repositoryID, &repositoryObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, false, apierrors.NewNotFound(configapi.TypeRepository.GroupResource(), repositoryID.Name)
		}
		return nil, false, apierrors.NewInternalError(fmt.Errorf("error getting repository %v: %w", repositoryID, err))
	}

	var parentPackage *engine.PackageRevision
	if newApiPkgRev.Spec.Parent != nil && newApiPkgRev.Spec.Parent.Name != "" {
		p, err := r.getRepoPkgRev(ctx, newApiPkgRev.Spec.Parent.Name)
		if err != nil {
			return nil, false, fmt.Errorf("cannot get parent package %q: %w", newApiPkgRev.Spec.Parent.Name, err)
		}
		parentPackage = p
	}

	if !isCreate {
		rev, err := r.cad.UpdatePackageRevision(ctx, &repositoryObj, oldRepoPkgRev, oldApiPkgRev.(*api.PackageRevision), newApiPkgRev, parentPackage)
		if err != nil {
			return nil, false, apierrors.NewInternalError(err)
		}

		updated, err := rev.GetPackageRevision(ctx)
		if err != nil {
			return nil, false, apierrors.NewInternalError(err)
		}

		return updated, false, nil
	} else {
		rev, err := r.cad.CreatePackageRevision(ctx, &repositoryObj, newApiPkgRev, parentPackage)
		if err != nil {
			klog.Infof("error creating package: %v", err)
			return nil, false, apierrors.NewInternalError(err)
		}
		createdApiPkgRev, err := rev.GetPackageRevision(ctx)
		if err != nil {
			return nil, false, apierrors.NewInternalError(err)
		}

		return createdApiPkgRev, true, nil
	}
}

// Common implementation of Package update logic.
func (r *packageCommon) updatePackage(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
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
		oldRuntimeObj = oldPackage.GetPackage()
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

	if err := r.validateUpdate(ctx, newRuntimeObj, oldRuntimeObj, isCreate, createValidation,
		updateValidation, "Package", name); err != nil {
		return nil, false, err
	}

	newObj, ok := newRuntimeObj.(*api.Package)
	if !ok {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("expected Package object, got %T", newRuntimeObj))
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
			return nil, false, apierrors.NewNotFound(configapi.TypeRepository.GroupResource(), repositoryID.Name)
		}
		return nil, false, apierrors.NewInternalError(fmt.Errorf("error getting repository %v: %w", repositoryID, err))
	}

	if !isCreate {
		rev, err := r.cad.UpdatePackage(ctx, &repositoryObj, oldPackage, oldRuntimeObj.(*api.Package), newObj)
		if err != nil {
			return nil, false, apierrors.NewInternalError(err)
		}

		updated := rev.GetPackage()

		return updated, false, nil
	} else {
		rev, err := r.cad.CreatePackage(ctx, &repositoryObj, newObj)
		if err != nil {
			klog.Infof("error creating package: %v", err)
			return nil, false, apierrors.NewInternalError(err)
		}

		created := rev.GetPackage()
		return created, true, nil
	}
}

func (r *packageCommon) validateDelete(ctx context.Context, deleteValidation rest.ValidateObjectFunc, obj runtime.Object,
	repoName string, ns string) (*configapi.Repository, error) {
	if deleteValidation != nil {
		err := deleteValidation(ctx, obj)
		if err != nil {
			klog.Infof("delete failed validation: %v", err)
			return nil, err
		}
	}
	repositoryName, err := ParseRepositoryName(repoName)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid name %q", repoName))
	}
	repositoryObj, err := r.getRepositoryObj(ctx, types.NamespacedName{Name: repositoryName, Namespace: ns})
	if err != nil {
		return nil, err
	}
	return repositoryObj, nil
}

func (r *packageCommon) validateUpdate(ctx context.Context, newRuntimeObj runtime.Object, oldRuntimeObj runtime.Object, create bool,
	createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, kind string, name string) error {
	r.updateStrategy.PrepareForUpdate(ctx, newRuntimeObj, oldRuntimeObj)

	if create {
		if createValidation != nil {
			err := createValidation(ctx, newRuntimeObj)
			if err != nil {
				klog.Infof("update failed create validation: %v", err)
				return err
			}
		}

		fieldErrors := r.createStrategy.Validate(ctx, newRuntimeObj)
		if len(fieldErrors) > 0 {
			return apierrors.NewInvalid(api.SchemeGroupVersion.WithKind(kind).GroupKind(), name, fieldErrors)
		}
	}

	if !create {
		if updateValidation != nil {
			err := updateValidation(ctx, newRuntimeObj, oldRuntimeObj)
			if err != nil {
				klog.Infof("update failed validation: %v", err)
				return err
			}
		}

		fieldErrors := r.updateStrategy.ValidateUpdate(ctx, newRuntimeObj, oldRuntimeObj)
		if len(fieldErrors) > 0 {
			return apierrors.NewInvalid(api.SchemeGroupVersion.WithKind(kind).GroupKind(), name, fieldErrors)
		}
	}

	r.updateStrategy.Canonicalize(newRuntimeObj)
	return nil
}
