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

package packagevariantset

import (
	"context"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha2"
)

func renderPackageVariantSpec(ctx context.Context, pvs *api.PackageVariantSet, repoList *configapi.RepositoryList,
	upstreamPR *porchapi.PackageRevision, downstream pvContext) (*pkgvarapi.PackageVariantSpec, error) {

	spec := &pkgvarapi.PackageVariantSpec{
		Upstream: pvs.Spec.Upstream,
		Downstream: &pkgvarapi.Downstream{
			Repo:    downstream.repo,
			Package: downstream.packageName,
		},
	}

	pvt := downstream.template
	if pvt == nil {
		return spec, nil
	}

	if pvt.Downstream != nil {
		if pvt.Downstream.Repo != nil && *pvt.Downstream.Repo != "" {
			spec.Downstream.Repo = *pvt.Downstream.Repo
		}
		if pvt.Downstream.Package != nil && *pvt.Downstream.Package != "" {
			spec.Downstream.Package = *pvt.Downstream.Package
		}
	}

	if pvt.AdoptionPolicy != nil {
		spec.AdoptionPolicy = *pvt.AdoptionPolicy
	}

	if pvt.DeletionPolicy != nil {
		spec.DeletionPolicy = *pvt.DeletionPolicy
	}

	spec.Labels = pvt.Labels
	spec.Annotations = pvt.Annotations

	if pvt.PackageContext != nil {
		spec.PackageContext = &pkgvarapi.PackageContext{
			Data:       pvt.PackageContext.Data,
			RemoveKeys: pvt.PackageContext.RemoveKeys,
		}
	}

	return spec, nil
}
