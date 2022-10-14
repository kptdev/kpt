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

package util

import (
	"context"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PackageAlreadyExists(ctx context.Context, c client.Client, repository, packageName, namespace string) (bool, error) {
	// only the first package revision can be created from init or clone, so
	// we need to check that the package doesn't already exist.
	packageRevisionList := porchapi.PackageRevisionList{}
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
