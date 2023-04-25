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

package v1alpha2

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	v1alpha1 "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha1"
)

func (src *PackageVariantSet) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.PackageVariantSet)

	dst.ObjectMeta = src.ObjectMeta

	if src.Spec.Upstream != nil {
		dst.Spec.Upstream = &v1alpha1.Upstream{
			Package: &v1alpha1.Package{
				Repo: src.Spec.Upstream.Repo,
				Name: src.Spec.Upstream.Package,
			},
			Revision: src.Spec.Upstream.Revision,
		}
	}

	// convert all the targets
	for _, t := range src.Spec.Targets {
		// Repositories map to Package targets
		for _, r := range t.Repositories {
			if len(r.PackageNames) == 0 {
				pkg := src.ObjectMeta.Name
				if src.Spec.Upstream != nil {
					pkg = src.Spec.Upstream.Package
				}
				dst.Spec.Targets = append(dst.Spec.Targets, v1alpha1.Target{
					Package: &v1alpha1.Package{
						Repo: r.Name,
						Name: pkg,
					},
				})
			}
			for _, pn := range r.PackageNames {
				dst.Spec.Targets = append(dst.Spec.Targets, v1alpha1.Target{
					Package: &v1alpha1.Package{
						Repo: r.Name,
						Name: pn,
					},
				})
			}
		}

		// RepositorySelector maps directly
		dst.Spec.Targets = append(dst.Spec.Targets, v1alpha1.Target{Repositories: t.RepositorySelector})

		// TODO: support this conversion
		// just fail object selectors for now
		if t.ObjectSelector != nil {
			return fmt.Errorf("conversion of object selector targets is not supported")
		}

		// TODO: support this conversion
		// just fail now (we should be able to convert some parts)
		if t.Template != nil {
			return fmt.Errorf("conversion of package variant template is not supported")
		}
	}

	return nil
}

func (dst *PackageVariantSet) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.PackageVariantSet)

	dst.ObjectMeta = src.ObjectMeta

	if src.Spec.Upstream != nil && src.Spec.Upstream.Package != nil {
		dst.Spec.Upstream = &pkgvarapi.Upstream{
			Repo:     src.Spec.Upstream.Package.Repo,
			Package:  src.Spec.Upstream.Package.Name,
			Revision: src.Spec.Upstream.Revision,
		}
	}

	return nil
}
