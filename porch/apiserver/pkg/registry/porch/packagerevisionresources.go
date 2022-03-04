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
	"github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
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

type packageRevisionResources struct {
	rest.TableConvertor

	packageCommon
}

var _ rest.Storage = &packageRevisionResources{}
var _ rest.Lister = &packageRevisionResources{}
var _ rest.Getter = &packageRevisionResources{}
var _ rest.Scoper = &packageRevisionResources{}
var _ rest.Updater = &packageRevisionResources{}

func (r *packageRevisionResources) New() runtime.Object {
	return &api.PackageRevisionResources{}
}

func (r *packageRevisionResources) NewList() runtime.Object {
	return &api.PackageRevisionResourcesList{}
}

func (r *packageRevisionResources) NamespaceScoped() bool {
	return true
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
func (r *packageRevisionResources) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	result := &api.PackageRevisionResourcesList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevisionResourcesList",
			APIVersion: api.SchemeGroupVersion.Identifier(),
		},
	}

	if err := r.packageCommon.listPackages(ctx, func(p repository.PackageRevision) error {
		item, err := p.GetResources(ctx)
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
func (r *packageRevisionResources) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	pkg, err := r.packageCommon.getPackage(ctx, name)
	if err != nil {
		return nil, err
	}

	obj, err := pkg.GetResources(ctx)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Update finds a resource in the storage and updates it. Some implementations
// may allow updates creates the object - they should set the created boolean
// to true.
func (r *packageRevisionResources) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, false, apierrors.NewBadRequest("namespace must be specified")
	}

	oldPackage, err := r.packageCommon.getPackage(ctx, name)
	if err != nil {
		return nil, false, err
	}

	oldObj, err := oldPackage.GetResources(ctx)
	if err != nil {
		klog.Infof("update failed to retrieve old object: %v", err)
		return nil, false, err
	}

	newRuntimeObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		klog.Infof("update failed to construct UpdatedObject: %v", err)
		return nil, false, err
	}
	newObj, ok := newRuntimeObj.(*api.PackageRevisionResources)
	if !ok {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("expected PackageRevisionResources object, got %T", newRuntimeObj))
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

	var repositoryObj v1alpha1.Repository
	repositoryID := types.NamespacedName{Namespace: ns, Name: nameTokens.RepositoryName}
	if err := r.coreClient.Get(ctx, repositoryID, &repositoryObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, false, apierrors.NewNotFound(api.PackageRevisionResourcesGVR.GroupResource(), repositoryID.Name)
		}
		return nil, false, apierrors.NewInternalError(fmt.Errorf("error getting repository %v: %w", repositoryID, err))
	}

	rev, err := r.cad.UpdatePackageResources(ctx, &repositoryObj, oldPackage, oldObj, newObj)
	if err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}

	created, err := rev.GetResources(ctx)
	if err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}
	return created, false, nil
}
