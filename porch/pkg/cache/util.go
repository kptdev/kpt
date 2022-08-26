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

package cache

import (
	"sort"
	"strings"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"golang.org/x/mod/semver"
	"k8s.io/klog/v2"
)

func toCachedPackageRevisionSlice(revisions []repository.PackageRevision) []*cachedPackageRevision {
	result := make([]*cachedPackageRevision, len(revisions))
	for i := range revisions {
		current := &cachedPackageRevision{
			PackageRevision:  revisions[i],
			isLatestRevision: false,
		}
		result[i] = current
	}
	identifyLatestRevisions(result)
	return result
}

func toCachedPackageSlice(packages []repository.Package) []*cachedPackage {
	result := make([]*cachedPackage, len(packages))
	for i := range packages {
		current := &cachedPackage{
			Package:               packages[i],
			latestPackageRevision: packages[i].GetLatestRevision(),
		}
		result[i] = current
	}
	return result
}

func identifyLatestRevisions(result []*cachedPackageRevision) {
	// Compute the latest among the different revisions of the same package.
	// The map is keyed by the package name; Values are the latest revision found so far.
	latest := map[string]*cachedPackageRevision{}
	for _, current := range result {
		current.isLatestRevision = false // Clear all values

		// Check if the current package revision is more recent than the one seen so far.
		// Only consider Published packages
		if current.Lifecycle() != v1alpha1.PackageRevisionLifecyclePublished {
			continue
		}

		currentKey := current.Key()
		if previous, ok := latest[currentKey.Package]; ok {
			previousKey := previous.Key()
			switch cmp := semver.Compare(currentKey.Revision, previousKey.Revision); {
			case cmp == 0:
				// Same revision.
				klog.Warningf("Encountered package revisions whose versions compare equal: %q, %q", currentKey, previousKey)
			case cmp < 0:
				// currentKey.Revision < previousKey.Revision; no change
			case cmp > 0:
				// currentKey.Revision > previousKey.Revision; update latest
				latest[currentKey.Package] = current
			}
		} else if semver.IsValid(currentKey.Revision) {
			// First revision of the specific package; candidate for the latest.
			latest[currentKey.Package] = current
		}
	}
	// Mark the winners as latest
	for _, v := range latest {
		v.isLatestRevision = true
	}
}

func toPackageRevisionSlice(cached []*cachedPackageRevision, filter repository.ListPackageRevisionFilter) []repository.PackageRevision {
	result := make([]repository.PackageRevision, 0, len(cached))
	for _, p := range cached {
		if filter.Matches(p) {
			result = append(result, p)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		ki, kl := result[i].Key(), result[j].Key()
		switch res := strings.Compare(ki.Package, kl.Package); {
		case res < 0:
			return true
		case res > 0:
			return false
		default:
			// Equal. Compare next element
		}
		switch res := strings.Compare(ki.Revision, kl.Revision); {
		case res < 0:
			return true
		case res > 0:
			return false
		default:
			// Equal. Compare next element
		}
		switch res := strings.Compare(string(result[i].Lifecycle()), string(result[j].Lifecycle())); {
		case res < 0:
			return true
		case res > 0:
			return false
		default:
			// Equal. Compare next element
		}

		return strings.Compare(result[i].KubeObjectName(), result[j].KubeObjectName()) < 0
	})
	return result
}

func toPackageSlice(cached []*cachedPackage, filter repository.ListPackageFilter) []repository.Package {
	result := make([]repository.Package, 0, len(cached))
	for _, p := range cached {
		if filter.Matches(p) {
			result = append(result, p)
		}
	}
	return result
}
