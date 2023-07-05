// Copyright 2023 The kpt Authors
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

package util

import (
	"context"
	"fmt"

	fnsdk "github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ResourceVersionAnnotation = "internal.kpt.dev/resource-version"
)

func PackageAlreadyExists(ctx context.Context, c client.Client, repository, packageName, namespace string) (bool, error) {
	// only the first package revision can be created from init or clone, so
	// we need to check that the package doesn't already exist.
	packageRevisionList := api.PackageRevisionList{}
	if err := c.List(ctx, &packageRevisionList, &client.ListOptions{
		Namespace: namespace,
	}); err != nil {
		return false, err
	}
	for _, pr := range packageRevisionList.Items {
		if pr.Spec.RepositoryName == repository && pr.Spec.PackageName == packageName {
			return true, nil
		}
	}
	return false, nil
}

func GetResourceFileKubeObject(prr *api.PackageRevisionResources, file, kind, name string) (*fnsdk.KubeObject, error) {
	if prr.Spec.Resources == nil {
		return nil, fmt.Errorf("nil resources found for PackageRevisionResources '%s/%s'", prr.Namespace, prr.Name)
	}

	if _, ok := prr.Spec.Resources[file]; !ok {
		return nil, fmt.Errorf("%q not found in PackageRevisionResources '%s/%s'", file, prr.Namespace, prr.Name)
	}

	ko, err := fnsdk.ParseKubeObject([]byte(prr.Spec.Resources[file]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q of PackageRevisionResources %s/%s: %w", file, prr.Namespace, prr.Name, err)
	}
	if kind != "" && ko.GetKind() != kind {
		return nil, fmt.Errorf("%q does not contain kind %q in PackageRevisionResources '%s/%s'", file, kind, prr.Namespace, prr.Name)
	}
	if name != "" && ko.GetName() != name {
		return nil, fmt.Errorf("%q does not contain resource named %q in PackageRevisionResources '%s/%s'", file, name, prr.Namespace, prr.Name)
	}

	return ko, nil
}

func GetResourceVersionAnnotation(prr *api.PackageRevisionResources) (string, error) {
	ko, err := GetResourceFileKubeObject(prr, "Kptfile", "Kptfile", "")

	if err != nil {
		return "", err
	}
	annotations := ko.GetAnnotations()
	rv, ok := annotations[ResourceVersionAnnotation]
	if !ok {
		rv = ""
	}
	return rv, nil
}

func AddResourceVersionAnnotation(prr *api.PackageRevisionResources) error {
	ko, err := GetResourceFileKubeObject(prr, "Kptfile", "Kptfile", "")
	if err != nil {
		return err
	}

	ko.SetAnnotation(ResourceVersionAnnotation, prr.GetResourceVersion())
	prr.Spec.Resources["Kptfile"] = ko.String()

	return nil
}

func RemoveResourceVersionAnnotation(prr *api.PackageRevisionResources) error {
	ko, err := GetResourceFileKubeObject(prr, "Kptfile", "Kptfile", "")
	if err != nil {
		return err
	}

	_, err = ko.RemoveNestedField("metadata", "annotations", ResourceVersionAnnotation)
	if err != nil {
		return err
	}
	prr.Spec.Resources["Kptfile"] = ko.String()

	return nil
}
