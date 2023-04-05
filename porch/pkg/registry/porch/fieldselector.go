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
	"fmt"

	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/selection"
)

// convertPackageRevisionFieldSelector is the schema conversion function for normalizing the the FieldSelector for PackageRevision
func convertPackageFieldSelector(label, value string) (internalLabel, internalValue string, err error) {
	switch label {
	case "metadata.name":
		return label, value, nil
	case "metadata.namespace":
		return label, value, nil
	case "spec.packageName", "spec.repository":
		return label, value, nil
	default:
		return "", "", fmt.Errorf("%q is not a known field selector", label)
	}
}

// convertPackageRevisionFieldSelector is the schema conversion function for normalizing the the FieldSelector for PackageRevision
func convertPackageRevisionFieldSelector(label, value string) (internalLabel, internalValue string, err error) {
	switch label {
	case "metadata.name":
		return label, value, nil
	case "metadata.namespace":
		return label, value, nil
	case "spec.revision", "spec.packageName", "spec.repository":
		return label, value, nil
	default:
		return "", "", fmt.Errorf("%q is not a known field selector", label)
	}
}

// convertPackageRevisionResourcesFieldSelector is the schema conversion function for normalizing the the FieldSelector for PackageRevisionResources
func convertPackageRevisionResourcesFieldSelector(label, value string) (internalLabel, internalValue string, err error) {
	return convertPackageRevisionFieldSelector(label, value)
}

// packageRevisionFilter filters packages, extending repository.ListPackageRevisionFilter
type packageRevisionFilter struct {
	repository.ListPackageRevisionFilter

	// Namespace filters by the namespace of the objects
	Namespace string

	// Repository restricts to repositories with the given name.
	Repository string
}

// packageFilter filters packages, extending repository.ListPackageFilter
type packageFilter struct {
	repository.ListPackageFilter

	// Namespace filters by the namespace of the objects
	Namespace string

	// Repository restricts to repositories with the given name.
	Repository string
}

// parsePackageFieldSelector parses client-provided fields.Selector into a packageFilter
func parsePackageFieldSelector(fieldSelector fields.Selector) (packageFilter, error) {
	var filter packageFilter

	if fieldSelector == nil {
		return filter, nil
	}

	requirements := fieldSelector.Requirements()
	for _, requirement := range requirements {

		switch requirement.Operator {
		case selection.Equals, selection.DoesNotExist:
			if requirement.Value == "" {
				return filter, apierrors.NewBadRequest(fmt.Sprintf("unsupported fieldSelector value %q for field %q with operator %q", requirement.Value, requirement.Field, requirement.Operator))
			}
		default:
			return filter, apierrors.NewBadRequest(fmt.Sprintf("unsupported fieldSelector operator %q for field %q", requirement.Operator, requirement.Field))
		}

		switch requirement.Field {
		case "metadata.name":
			filter.KubeObjectName = requirement.Value

		case "metadata.namespace":
			filter.Namespace = requirement.Value

		case "spec.packageName":
			filter.Package = requirement.Value
		case "spec.repository":
			filter.Repository = requirement.Value

		default:
			return filter, apierrors.NewBadRequest(fmt.Sprintf("unknown fieldSelector field %q", requirement.Field))
		}
	}

	return filter, nil
}

// parsePackageRevisionFieldSelector parses client-provided fields.Selector into a packageRevisionFilter
func parsePackageRevisionFieldSelector(fieldSelector fields.Selector) (packageRevisionFilter, error) {
	var filter packageRevisionFilter

	if fieldSelector == nil {
		return filter, nil
	}

	requirements := fieldSelector.Requirements()
	for _, requirement := range requirements {

		switch requirement.Operator {
		case selection.Equals, selection.DoesNotExist:
			if requirement.Value == "" {
				return filter, apierrors.NewBadRequest(fmt.Sprintf("unsupported fieldSelector value %q for field %q with operator %q", requirement.Value, requirement.Field, requirement.Operator))
			}
		default:
			return filter, apierrors.NewBadRequest(fmt.Sprintf("unsupported fieldSelector operator %q for field %q", requirement.Operator, requirement.Field))
		}

		switch requirement.Field {
		case "metadata.name":
			filter.KubeObjectName = requirement.Value

		case "metadata.namespace":
			filter.Namespace = requirement.Value

		case "spec.revision":
			filter.Revision = requirement.Value
		case "spec.packageName":
			filter.Package = requirement.Value
		case "spec.repository":
			filter.Repository = requirement.Value

		default:
			return filter, apierrors.NewBadRequest(fmt.Sprintf("unknown fieldSelector field %q", requirement.Field))
		}
	}

	return filter, nil
}

// parsePackageRevisionResourcesFieldSelector parses client-provided fields.Selector into a packageRevisionFilter
func parsePackageRevisionResourcesFieldSelector(fieldSelector fields.Selector) (packageRevisionFilter, error) {
	// TOOD: This is a little weird, because we don't have the same fields on PackageRevisionResources.
	// But we probably should have the key fields
	return parsePackageRevisionFieldSelector(fieldSelector)
}
