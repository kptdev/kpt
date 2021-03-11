// Copyright 2021 Google LLC
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

//nolint:dupl
package update

import (
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	"github.com/stretchr/testify/assert"
)

func TestUpdate_ResourceMerge(t *testing.T) {
	testCases := map[string]struct {
		origin         *pkgbuilder.RootPkg
		local          *pkgbuilder.RootPkg
		updated        *pkgbuilder.RootPkg
		relPackagePath string
		isRoot         bool
		expected       *pkgbuilder.RootPkg
	}{
		"updates local subpackages": {
			origin: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource),
				),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/GoogleContainerTools/kpt", "/", "master", "resource-merge").
						WithUpstreamLock(),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource),
				),
			updated: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithResource(pkgbuilder.ConfigMapResource),
				),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/GoogleContainerTools/kpt", "/", "master", "resource-merge").
						WithUpstreamLock(),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		"doesn't update remote subpackages": {
			origin: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream("github.com/GoogleContainerTools/kpt", "/", "main", "resource-merge"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/GoogleContainerTools/kpt", "/", "master", "resource-merge").
						WithUpstreamLock(),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream("github.com/GoogleContainerTools/kpt", "/", "main", "resource-merge"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			updated: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream("github.com/GoogleContainerTools/kpt", "/", "main", "resource-merge"),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/GoogleContainerTools/kpt", "/", "master", "resource-merge").
						WithUpstreamLock(),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream("github.com/GoogleContainerTools/kpt", "/", "main", "resource-merge"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
		},
		"doesn't update the Kptfile": {
			origin: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/GoogleContainerTools/kpt", "/", "main", "resource-merge"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/GoogleContainerTools/kpt", "/", "master", "resource-merge").
						WithUpstreamLock(),
				).
				WithResource(pkgbuilder.DeploymentResource),
			updated: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/GoogleContainerTools/kpt", "/", "v1.0", "resource-merge"),
				).
				WithResource(pkgbuilder.ConfigMapResource),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/GoogleContainerTools/kpt", "/", "master", "resource-merge").
						WithUpstreamLock(),
				).
				WithResource(pkgbuilder.ConfigMapResource),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			repoPaths := map[string]string{}
			origin := pkgbuilder.ExpandPkg(t, tc.origin, repoPaths)
			local := pkgbuilder.ExpandPkg(t, tc.local, repoPaths)
			updated := pkgbuilder.ExpandPkg(t, tc.updated, repoPaths)
			expected := pkgbuilder.ExpandPkg(t, tc.expected, repoPaths)

			updater := &FastForwardUpdater{}

			err := updater.Update(UpdateOptions{
				RelPackagePath: tc.relPackagePath,
				OriginPath:     origin,
				LocalPath:      local,
				UpdatedPath:    updated,
				IsRoot:         tc.isRoot,
			})
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			testutil.KptfileAwarePkgEqual(t, local, expected)
		})
	}
}
