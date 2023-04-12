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

	"k8s.io/apimachinery/pkg/runtime/schema"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/engine"
	"go.opentelemetry.io/otel/trace"
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

func (r *packageRevisionResources) Destroy() {}

func (r *packageRevisionResources) NewList() runtime.Object {
	return &api.PackageRevisionResourcesList{}
}

func (r *packageRevisionResources) NamespaceScoped() bool {
	return true
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
func (r *packageRevisionResources) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	ctx, span := tracer.Start(ctx, "packageRevisionResources::List", trace.WithAttributes())
	defer span.End()

	result := &api.PackageRevisionResourcesList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevisionResourcesList",
			APIVersion: api.SchemeGroupVersion.Identifier(),
		},
	}

	filter, err := parsePackageRevisionResourcesFieldSelector(options.FieldSelector)
	if err != nil {
		return nil, err
	}

	if err := r.packageCommon.listPackageRevisions(ctx, filter, options.LabelSelector, func(p *engine.PackageRevision) error {
		apiPkgResources, err := p.GetResources(ctx)
		if err != nil {
			return err
		}
		result.Items = append(result.Items, *apiPkgResources)
		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

// Get implements the Getter interface
func (r *packageRevisionResources) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ctx, span := tracer.Start(ctx, "packageRevisionResources::Get", trace.WithAttributes())
	defer span.End()

	pkg, err := r.packageCommon.getRepoPkgRev(ctx, name)
	if err != nil {
		return nil, err
	}

	apiPkgResources, err := pkg.GetResources(ctx)
	if err != nil {
		return nil, err
	}
	return apiPkgResources, nil
}

// Update finds a resource in the storage and updates it. Some implementations
// may allow updates creates the object - they should set the created boolean
// to true.
func (r *packageRevisionResources) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	ctx, span := tracer.Start(ctx, "packageRevisionResources::Update", trace.WithAttributes())
	defer span.End()

	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, false, apierrors.NewBadRequest("namespace must be specified")
	}

	oldRepoPkgRev, err := r.packageCommon.getRepoPkgRev(ctx, name)
	if err != nil {
		return nil, false, err
	}

	oldApiPkgRevResources, err := oldRepoPkgRev.GetResources(ctx)
	if err != nil {
		klog.Infof("update failed to retrieve old object: %v", err)
		return nil, false, err
	}

	newRuntimeObj, err := objInfo.UpdatedObject(ctx, oldApiPkgRevResources)
	if err != nil {
		klog.Infof("update failed to construct UpdatedObject: %v", err)
		return nil, false, err
	}
	newObj, ok := newRuntimeObj.(*api.PackageRevisionResources)
	if !ok {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("expected PackageRevisionResources object, got %T", newRuntimeObj))
	}

	if updateValidation != nil {
		err := updateValidation(ctx, newObj, oldApiPkgRevResources)
		if err != nil {
			klog.Infof("update failed validation: %v", err)
			return nil, false, err
		}
	}

	repositoryName, err := ParseRepositoryName(name)
	if err != nil {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("invalid name %q", name))
	}

	var repositoryObj v1alpha1.Repository
	repositoryID := types.NamespacedName{Namespace: ns, Name: repositoryName}
	if err := r.coreClient.Get(ctx, repositoryID, &repositoryObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, false, apierrors.NewNotFound(schema.GroupResource(api.PackageRevisionResourcesGVR.GroupResource()), repositoryID.Name)
		}
		return nil, false, apierrors.NewInternalError(fmt.Errorf("error getting repository %v: %w", repositoryID, err))
	}

	rev, renderStatus, err := r.cad.UpdatePackageResources(ctx, &repositoryObj, oldRepoPkgRev, oldApiPkgRevResources, newObj)
	if err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}

	created, err := rev.GetResources(ctx)
	if err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}
	if renderStatus != nil {
		created.Status.RenderStatus = *renderStatus
	}

	return created, false, nil
}
