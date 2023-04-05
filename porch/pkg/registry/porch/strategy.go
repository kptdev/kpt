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
	"strings"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// PackageRevisions Update Strategy

type packageRevisionStrategy struct{}

var _ SimpleRESTUpdateStrategy = packageRevisionStrategy{}

func (s packageRevisionStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (s packageRevisionStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	allErrs := field.ErrorList{}
	oldRevision := old.(*api.PackageRevision)
	newRevision := obj.(*api.PackageRevision)

	// Verify that the new lifecycle value is valid.
	switch lifecycle := newRevision.Spec.Lifecycle; lifecycle {
	case "", api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed, api.PackageRevisionLifecyclePublished, api.PackageRevisionLifecycleDeletionProposed:
		// valid
	default:
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "lifecycle"), lifecycle, fmt.Sprintf("value can be only updated to %s",
			strings.Join([]string{
				string(api.PackageRevisionLifecycleDraft),
				string(api.PackageRevisionLifecycleProposed),
				string(api.PackageRevisionLifecyclePublished),
				string(api.PackageRevisionLifecycleDeletionProposed),
			}, ",")),
		))
	}

	switch lifecycle := oldRevision.Spec.Lifecycle; lifecycle {
	case "", api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed:
		// Packages in a draft or proposed state can only be updated to draft or proposed.
		newLifecycle := newRevision.Spec.Lifecycle
		if !(newLifecycle == api.PackageRevisionLifecycleDraft ||
			newLifecycle == api.PackageRevisionLifecycleProposed ||
			newLifecycle == "") {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "lifecycle"), lifecycle, fmt.Sprintf("value can be only updated to %s",
				strings.Join([]string{
					string(api.PackageRevisionLifecycleDraft),
					string(api.PackageRevisionLifecycleProposed),
				}, ",")),
			))
		}
	case api.PackageRevisionLifecyclePublished, api.PackageRevisionLifecycleDeletionProposed:
		// We don't allow any updates to the spec for packagerevision that have been published. That includes updates of the lifecycle. But
		// we allow updates to metadata and status. The only exception is that the lifecycle
		// can change between Published and DeletionProposed and vice versa.
		newLifecycle := newRevision.Spec.Lifecycle
		if api.LifecycleIsPublished(newLifecycle) {
			// copy the lifecycle value over before calling reflect.DeepEqual to allow comparison
			// of all other fields without error
			newRevision.Spec.Lifecycle = oldRevision.Spec.Lifecycle
		}
		if !equality.Semantic.DeepEqual(oldRevision.Spec, newRevision.Spec) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), newRevision.Spec, fmt.Sprintf("spec can only update package with lifecycle value one of %s",
				strings.Join([]string{
					string(api.PackageRevisionLifecycleDraft),
					string(api.PackageRevisionLifecycleProposed),
				}, ",")),
			))
		}
		newRevision.Spec.Lifecycle = newLifecycle
	default:
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "lifecycle"), lifecycle, fmt.Sprintf("can only update package with lifecycle value one of %s",
			strings.Join([]string{
				string(api.PackageRevisionLifecycleDraft),
				string(api.PackageRevisionLifecycleProposed),
				string(api.PackageRevisionLifecyclePublished),
				string(api.PackageRevisionLifecycleDeletionProposed),
			}, ",")),
		))
	}

	return allErrs
}

func (s packageRevisionStrategy) Canonicalize(obj runtime.Object) {
	pr := obj.(*api.PackageRevision)
	if pr.Spec.Lifecycle == "" {
		// Set default
		pr.Spec.Lifecycle = api.PackageRevisionLifecycleDraft
	}
}

var _ SimpleRESTCreateStrategy = packageRevisionStrategy{}

// Validate returns an ErrorList with validation errors or nil.  Validate
// is invoked after default fields in the object have been filled in
// before the object is persisted.  This method should not mutate the
// object.
func (s packageRevisionStrategy) Validate(ctx context.Context, runtimeObj runtime.Object) field.ErrorList {
	allErrs := field.ErrorList{}

	obj := runtimeObj.(*api.PackageRevision)

	switch lifecycle := obj.Spec.Lifecycle; lifecycle {
	case "", api.PackageRevisionLifecycleDraft:
		// valid

	default:
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "lifecycle"), lifecycle, fmt.Sprintf("value can be only created as %s",
			strings.Join([]string{
				string(api.PackageRevisionLifecycleDraft),
			}, ",")),
		))
	}

	return allErrs
}

// Package Update Strategy

type packageStrategy struct{}

var _ SimpleRESTUpdateStrategy = packageStrategy{}

func (s packageStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (s packageStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return nil
}

func (s packageStrategy) Canonicalize(obj runtime.Object) {
}

var _ SimpleRESTCreateStrategy = packageStrategy{}

// Validate returns an ErrorList with validation errors or nil.  Validate
// is invoked after default fields in the object have been filled in
// before the object is persisted.  This method should not mutate the
// object.
func (s packageStrategy) Validate(ctx context.Context, runtimeObj runtime.Object) field.ErrorList {
	return nil
}
