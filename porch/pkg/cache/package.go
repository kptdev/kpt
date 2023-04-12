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

package cache

import "github.com/GoogleContainerTools/kpt/porch/pkg/repository"

// We take advantage of the cache having a global view of all the packages
// in a repository and compute the latest package revision in the cache
// rather than add another level of caching in the repositories themselves.
// This also reuses the revision comparison code and ensures same behavior
// between Git and OCI.

var _ repository.Package = &cachedPackage{}

type cachedPackage struct {
	repository.Package
	latestPackageRevision string
}

func (c *cachedPackage) GetLatestRevision() string {
	if c.latestPackageRevision != "" {
		return c.latestPackageRevision
	}
	return c.Package.GetLatestRevision()
}
