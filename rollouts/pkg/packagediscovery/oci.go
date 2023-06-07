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

package packagediscovery

import (
	"context"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
)

// getOCIPackages discovers OCI packages for OCI config.
// TODO(droot): Support variants discovery in the future.
func (d *PackageDiscovery) getOCIPackages(ctx context.Context, config gitopsv1alpha1.PackagesConfig) ([]DiscoveredPackage, error) {
	var discoveredPackages []DiscoveredPackage

	oci := config.OciSource

	discoveredPackages = append(discoveredPackages, DiscoveredPackage{
		Directory: oci.Selector.Dir,
		OciRepo: &OCIRepo{
			Image: oci.Selector.Image,
		},
	})
	return discoveredPackages, nil
}
