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

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
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

type packages struct {
	packageCommon
	rest.TableConvertor
}

var _ rest.Storage = &packages{}
var _ rest.Lister = &packages{}
var _ rest.Getter = &packages{}
var _ rest.Scoper = &packages{}
var _ rest.Creater = &packages{}
var _ rest.Updater = &packages{}
var _ rest.GracefulDeleter = &packages{}

func (r *packages) New() runtime.Object {
	return &api.Package{}
}

func (r *packages) Destroy() {}

func (r *packages) NewList() runtime.Object {
	return &api.PackageList{}
}

func (r *packages) NamespaceScoped() bool {
	return true
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
func (r *packages) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	ctx, span := tracer.Start(ctx, "packages::List", trace.WithAttributes())
	defer span.End()

	result := &api.PackageList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageList",
			APIVersion: api.SchemeGroupVersion.Identifier(),
		},
	}

	filter, err := parsePackageFieldSelector(options.FieldSelector)
	if err != nil {
		return nil, err
	}

	if err := r.packageCommon.listPackages(ctx, filter, func(p *engine.Package) error {
		item := p.GetPackage()
		result.Items = append(result.Items, *item)
		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

// Get implements the Getter interface
func (r *packages) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ctx, span := tracer.Start(ctx, "packages::Get", trace.WithAttributes())
	defer span.End()

	pkg, err := r.getPackage(ctx, name)
	if err != nil {
		return nil, err
	}

	obj := pkg.GetPackage()
	return obj, nil
}

// Create implements the Creater interface.
func (r *packages) Create(ctx context.Context, runtimeObject runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	ctx, span := tracer.Start(ctx, "packages::Create", trace.WithAttributes())
	defer span.End()

	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, apierrors.NewBadRequest("namespace must be specified")
	}

	obj, ok := runtimeObject.(*api.Package)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected Package object, got %T", runtimeObject))
	}

	// TODO: Accpept some form of client-provided name, for example using GenerateName
	// and figure out where we can store it (in Kptfile?). Porch can then append unique
	// suffix to the names while respecting client-provided value as well.
	if obj.Name != "" {
		klog.Warningf("Client provided metadata.name %q", obj.Name)
	}

	repositoryName := obj.Spec.RepositoryName
	if repositoryName == "" {
		return nil, apierrors.NewBadRequest("spec.repository is required")
	}

	repositoryObj, err := r.packageCommon.getRepositoryObj(ctx, types.NamespacedName{Name: repositoryName, Namespace: ns})
	if err != nil {
		return nil, err
	}

	fieldErrors := r.createStrategy.Validate(ctx, runtimeObject)
	if len(fieldErrors) > 0 {
		return nil, apierrors.NewInvalid(api.SchemeGroupVersion.WithKind("Package").GroupKind(), obj.Name, fieldErrors)
	}

	rev, err := r.cad.CreatePackage(ctx, repositoryObj, obj)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	created := rev.GetPackage()
	return created, nil
}

// Update implements the Updater interface.

// Update finds a resource in the storage and updates it. Some implementations
// may allow updates creates the object - they should set the created boolean
// to true.
func (r *packages) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	ctx, span := tracer.Start(ctx, "packages::Update", trace.WithAttributes())
	defer span.End()

	return r.packageCommon.updatePackage(ctx, name, objInfo, createValidation, updateValidation, forceAllowCreate, options)
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
func (r *packages) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	ctx, span := tracer.Start(ctx, "packages::Delete", trace.WithAttributes())
	defer span.End()

	// TODO: Verify options are empty?

	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, false, apierrors.NewBadRequest("namespace must be specified")
	}

	oldPackage, err := r.packageCommon.getPackage(ctx, name)
	if err != nil {
		return nil, false, err
	}

	oldObj := oldPackage.GetPackage()
	repositoryObj, err := r.packageCommon.validateDelete(ctx, deleteValidation, oldObj, name, ns)
	if err != nil {
		return nil, false, err
	}

	if err := r.cad.DeletePackage(ctx, repositoryObj, oldPackage); err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}

	// TODO: Should we do an async delete?
	return oldObj, true, nil
}
