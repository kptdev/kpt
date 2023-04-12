// Copyright 2021 The kpt Authors
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
package update_test

import (
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	. "github.com/GoogleContainerTools/kpt/internal/util/update"
	"github.com/stretchr/testify/assert"
)

func TestUpdate_Replace(t *testing.T) {
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
						WithUpstream(kptRepo, "/", "master", "force-delete-replace").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
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
						WithUpstream(kptRepo, "/", "master", "force-delete-replace").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
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
								WithUpstream(kptRepo, "/", "main", "force-delete-replace"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "force-delete-replace").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream(kptRepo, "/", "main", "force-delete-replace"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			updated: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream(kptRepo, "/", "main", "force-delete-replace"),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "force-delete-replace").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream(kptRepo, "/", "main", "force-delete-replace"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
		},
		"doesn't update the Kptfile": {
			origin: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "main", "force-delete-replace"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "force-delete-replace").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			updated: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "v1.0", "force-delete-replace"),
				).
				WithResource(pkgbuilder.ConfigMapResource),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "force-delete-replace").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.ConfigMapResource),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			repos := testutil.EmptyReposInfo
			origin := tc.origin.ExpandPkg(t, repos)
			local := tc.local.ExpandPkg(t, repos)
			updated := tc.updated.ExpandPkg(t, repos)
			expected := tc.expected.ExpandPkg(t, repos)

			updater := &ReplaceUpdater{}

			err := updater.Update(Options{
				RelPackagePath: tc.relPackagePath,
				OriginPath:     filepath.Join(origin, tc.relPackagePath),
				LocalPath:      filepath.Join(local, tc.relPackagePath),
				UpdatedPath:    filepath.Join(updated, tc.relPackagePath),
				IsRoot:         tc.isRoot,
			})
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			testutil.KptfileAwarePkgEqual(t, local, expected, false)
		})
	}
}
