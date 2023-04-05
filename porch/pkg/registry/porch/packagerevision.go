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
	"go.opentelemetry.io/otel"
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

var tracer = otel.Tracer("apiserver")

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
var _ rest.Watcher = &packageRevisions{}

func (r *packageRevisions) New() runtime.Object {
	return &api.PackageRevision{}
}

func (r *packageRevisions) Destroy() {}

func (r *packageRevisions) NewList() runtime.Object {
	return &api.PackageRevisionList{}
}

func (r *packageRevisions) NamespaceScoped() bool {
	return true
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
func (r *packageRevisions) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	ctx, span := tracer.Start(ctx, "packageRevisions::List", trace.WithAttributes())
	defer span.End()

	result := &api.PackageRevisionList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevisionList",
			APIVersion: api.SchemeGroupVersion.Identifier(),
		},
	}

	filter, err := parsePackageRevisionFieldSelector(options.FieldSelector)
	if err != nil {
		return nil, err
	}

	if err := r.packageCommon.listPackageRevisions(ctx, filter, options.LabelSelector, func(p *engine.PackageRevision) error {
		item, err := p.GetPackageRevision(ctx)
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
	ctx, span := tracer.Start(ctx, "packageRevisions::Get", trace.WithAttributes())
	defer span.End()

	repoPkgRev, err := r.getRepoPkgRev(ctx, name)
	if err != nil {
		return nil, err
	}

	apiPkgRev, err := repoPkgRev.GetPackageRevision(ctx)
	if err != nil {
		return nil, err
	}

	return apiPkgRev, nil
}

// Create implements the Creater interface.
func (r *packageRevisions) Create(ctx context.Context, runtimeObject runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	ctx, span := tracer.Start(ctx, "packageRevisions::Create", trace.WithAttributes())
	defer span.End()

	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, apierrors.NewBadRequest("namespace must be specified")
	}

	newApiPkgRev, ok := runtimeObject.(*api.PackageRevision)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected PackageRevision object, got %T", runtimeObject))
	}

	// TODO: Accept some form of client-provided name, for example using GenerateName
	// and figure out where we can store it (in Kptfile?). Porch can then append unique
	// suffix to the names while respecting client-provided value as well.
	if newApiPkgRev.Name != "" {
		klog.Warningf("Client provided metadata.name %q", newApiPkgRev.Name)
	}

	repositoryName := newApiPkgRev.Spec.RepositoryName
	if repositoryName == "" {
		return nil, apierrors.NewBadRequest("spec.repositoryName is required")
	}

	repositoryObj, err := r.packageCommon.getRepositoryObj(ctx, types.NamespacedName{Name: repositoryName, Namespace: ns})
	if err != nil {
		return nil, err
	}

	fieldErrors := r.createStrategy.Validate(ctx, runtimeObject)
	if len(fieldErrors) > 0 {
		return nil, apierrors.NewInvalid(api.SchemeGroupVersion.WithKind("PackageRevision").GroupKind(), newApiPkgRev.Name, fieldErrors)
	}

	var parentPackage *engine.PackageRevision
	if newApiPkgRev.Spec.Parent != nil && newApiPkgRev.Spec.Parent.Name != "" {
		p, err := r.packageCommon.getRepoPkgRev(ctx, newApiPkgRev.Spec.Parent.Name)
		if err != nil {
			return nil, fmt.Errorf("cannot get parent package %q: %w", newApiPkgRev.Spec.Parent.Name, err)
		}
		parentPackage = p
	}

	createdRepoPkgRev, err := r.cad.CreatePackageRevision(ctx, repositoryObj, newApiPkgRev, parentPackage)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	createdApiPkgRev, err := createdRepoPkgRev.GetPackageRevision(ctx)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}

	return createdApiPkgRev, nil
}

// Update implements the Updater interface.

// Update finds a resource in the storage and updates it. Some implementations
// may allow updates creates the object - they should set the created boolean
// to true.
func (r *packageRevisions) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	ctx, span := tracer.Start(ctx, "packageRevisions::Update", trace.WithAttributes())
	defer span.End()

	return r.packageCommon.updatePackageRevision(ctx, name, objInfo, createValidation, updateValidation, forceAllowCreate, options)
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
	ctx, span := tracer.Start(ctx, "packageRevisions::Delete", trace.WithAttributes())
	defer span.End()

	ns, namespaced := genericapirequest.NamespaceFrom(ctx)
	if !namespaced {
		return nil, false, apierrors.NewBadRequest("namespace must be specified")
	}

	repoPkgRev, err := r.packageCommon.getRepoPkgRev(ctx, name)
	if err != nil {
		return nil, false, err
	}

	apiPkgRev, err := repoPkgRev.GetPackageRevision(ctx)
	if err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}

	repositoryObj, err := r.packageCommon.validateDelete(ctx, deleteValidation, apiPkgRev, name, ns)
	if err != nil {
		return nil, false, err
	}

	if err := r.cad.DeletePackageRevision(ctx, repositoryObj, repoPkgRev); err != nil {
		return nil, false, apierrors.NewInternalError(err)
	}

	// TODO: Should we do an async delete?
	return apiPkgRev, true, nil
}
