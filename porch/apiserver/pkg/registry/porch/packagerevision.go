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
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"
)

type packageRevisions struct {
	packageCommon

	rest.TableConvertor
}

var _ rest.Storage = &packageRevisions{}
var _ rest.Lister = &packageRevisions{}
var _ rest.Getter = &packageRevisions{}
var _ rest.Scoper = &packageRevisions{}
var _ rest.Creater = &packageRevisions{}
var _ rest.Updater = &packageRevisions{}
var _ rest.GracefulDeleter = &packageRevisions{}

func (r *packageRevisions) New() runtime.Object {
	return &api.PackageRevision{}
}

func (r *packageRevisions) NewList() runtime.Object {
	return &api.PackageRevisionList{}
}

func (r *packageRevisions) NamespaceScoped() bool {
	return true
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
func (r *packageRevisions) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	result := &api.PackageRevisionList{}

	if err := r.packageCommon.listPackages(ctx, func(p repository.PackageRevision) error {
		item, err := p.GetPackageRevision()
		if err != nil {
			return err
		}
		result.Items = append(result.Items, *item)
		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

// Get implements the Getter interface
func (r *packageRevisions) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	pkg, err := r.packageCommon.getPackage(ctx, name)
	if err != nil {
		return nil, err
	}

	obj, err := pkg.GetPackageRevision()
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Create implements the Creater interface.
func (r *packageRevisions) Create(ctx context.Context, runtimeObject runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, apierrors.NewBadRequest("namespace must be specified")
	}

	obj, ok := runtimeObject.(*api.PackageRevision)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected PackageRevision object, got %T", runtimeObject))
	}

	expectedName := obj.Spec.RepositoryName + ":" + obj.Spec.PackageName + ":" + obj.Spec.Revision
	name := obj.Name
	if name == "" {
		name = expectedName
	}

	if name != expectedName {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("name should be %q", expectedName))
	}

	nameTokens, err := ParseName(name)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid name %q", name))
	}

	var repositoryObj configapi.Repository
	repositoryID := types.NamespacedName{Namespace: ns, Name: nameTokens.RepositoryName}
	if err := r.coreClient.Get(ctx, repositoryID, &repositoryObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apierrors.NewNotFound(configapi.KindRepository.GroupResource(), repositoryID.Name)
		}
		return nil, apierrors.NewInternalError(fmt.Errorf("error getting repository %v: %w", repositoryID, err))
	}

	rev, err := r.cad.CreatePackageRevision(ctx, &repositoryObj, obj)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	created, err := rev.GetPackageRevision()
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	return created, nil
}

// Update implements the Updater interface.

// Update finds a resource in the storage and updates it. Some implementations
// may allow updates creates the object - they should set the created boolean
// to true.
func (r *packageRevisions) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	// TODO: Is this all boilerplate??

	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, false, apierrors.NewBadRequest("namespace must be specified")
	}

	oldPackage, err := r.packageCommon.getPackage(ctx, name)
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
	newObj, ok := newRuntimeObj.(*api.PackageRevision)
	if !ok {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("expected PackageRevision object, got %T", newRuntimeObj))
	}

	if updateValidation != nil {
		err := updateValidation(ctx, newObj, oldObj)
		if err != nil {
			klog.Infof("update failed validation: %v", err)
			return nil, false, err
		}
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

// Delete implements the GracefulDeleter interface.
// Delete finds a resource in the storage and deletes it.
// The delete attempt is validated by the deleteValidation first.
// If options are provided, the resource will attempt to honor them or return an invalid
// request error.
// Although it can return an arbitrary error value, IsNotFound(err) is true for the
// returned error value err when the specified resource is not found.
// Delete *may* return the object that was deleted, or a status object indicating additional
// information about deletion.
// It also returns a boolean which is set to true if the resource was instantly
// deleted or false if it will be deleted asynchronously.
func (r *packageRevisions) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, false, apierrors.NewBadRequest("namespace must be specified")
	}

	oldPackage, err := r.packageCommon.getPackage(ctx, name)
	if err != nil {
		return nil, false, err
	}

	oldObj, err := oldPackage.GetPackageRevision()
	if err != nil {
		klog.Infof("update failed to retrieve old object: %v", err)
		return nil, false, err
	}

	if deleteValidation != nil {
		err := deleteValidation(ctx, oldObj)
		if err != nil {
			klog.Infof("delete failed validation: %v", err)
			return nil, false, err
		}
	}

	// TODO: Verify options are empty?

	nameTokens, err := ParseName(name)
	if err != nil {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("invalid name %q", name))
	}

	var repositoryObj configapi.Repository
	repositoryID := types.NamespacedName{Namespace: ns, Name: nameTokens.RepositoryName}
	if err := r.coreClient.Get(ctx, repositoryID, &repositoryObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, false, apierrors.NewNotFound(configapi.KindRepository.GroupResource(), repositoryID.Name)
		}
		return nil, false, apierrors.NewInternalError(fmt.Errorf("error getting repository %v: %w", repositoryID, err))
	}

	if err := r.cad.DeletePackageRevision(ctx, &repositoryObj, oldPackage); err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}

	// TODO: Should we do an async delete?
	return oldObj, true, nil
}
