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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kptdev/kpt/internal/testutil"
	"github.com/kptdev/kpt/internal/testutil/pkgbuilder"
	"github.com/kptdev/kpt/pkg/lib/update"
	"github.com/kptdev/kpt/pkg/lib/update/updatetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
						WithUpstream(kptRepo, "/", "master", "resource-merge").
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
						WithUpstream(kptRepo, "/", "master", "resource-merge").
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
								WithUpstream(kptRepo, "/", "main", "resource-merge"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "resource-merge").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream(kptRepo, "/", "main", "resource-merge"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			updated: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream(kptRepo, "/", "main", "resource-merge"),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "resource-merge").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream(kptRepo, "/", "main", "resource-merge"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
		},
		"doesn't update the Kptfile if package is the root": {
			origin: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "main", "resource-merge"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "resource-merge").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			updated: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "v1.0", "resource-merge"),
				).
				WithResource(pkgbuilder.ConfigMapResource),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "resource-merge").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.ConfigMapResource),
		},
		"updates the Kptfile if package is not the root and local hasn't changed from origin": {
			origin: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "master", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "master", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			updated: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "v1.0", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "v1.0", "def456"),
				).
				WithResource(pkgbuilder.ConfigMapResource),
			relPackagePath: "/",
			isRoot:         false,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "v1.0", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "v1.0", "def456"),
				).
				WithResource(pkgbuilder.ConfigMapResource),
		},
		"does not update the local package at all if not root and upstream info is changed on local": {
			origin: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "main", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "main", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "feature-branch", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "feature-branch", "def456"),
				).
				WithResource(pkgbuilder.SecretResource),
			updated: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "v1.0", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "v1.0", "qwerty"),
				).
				WithResource(pkgbuilder.ConfigMapResource),
			relPackagePath: "/",
			isRoot:         false,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "feature-branch", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "feature-branch", "def456"),
				).
				WithResource(pkgbuilder.SecretResource),
		},
		"does not remove a file from local if it has local changes": {
			origin: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.SecretResource).
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "feature-branch", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "feature-branch", "def456"),
				).
				WithResource(pkgbuilder.SecretResource).
				WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("5", "spec", "replicas")),
			updated: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.SecretResource),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "feature-branch", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "feature-branch", "def456"),
				).
				WithResource(pkgbuilder.SecretResource).
				WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("5", "spec", "replicas")),
		},
		"does not re-add files from upstream if deleted from local": {
			origin: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.SecretResource).
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "feature-branch", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "feature-branch", "def456"),
				).
				WithResource(pkgbuilder.SecretResource),
			updated: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.SecretResource).
				WithResource(pkgbuilder.DeploymentResource),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream("github.com/kptdev/kpt", "/", "feature-branch", "resource-merge").
						WithUpstreamLock("github.com/kptdev/kpt", "/", "feature-branch", "def456"),
				).
				WithResource(pkgbuilder.SecretResource),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			repos := testutil.EmptyReposInfo
			origin := tc.origin.ExpandPkg(t, repos)
			local := tc.local.ExpandPkg(t, repos)
			updated := tc.updated.ExpandPkg(t, repos)
			expected := tc.expected.ExpandPkg(t, repos)

			updater := &update.ResourceMergeUpdater{}

			err := updater.Update(updatetypes.Options{
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

func TestKeepTaggedNull(t *testing.T) {
	updater := update.ResourceMergeUpdater{}

	testdata, err := filepath.Abs(filepath.Join(".", "testdata", "keep-tagged-null"))
	require.NoError(t, err)

	tmpDir := t.TempDir()
	newDest := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(newDest, 0755)
	require.NoError(t, err)

	oldDestFiles, err := os.ReadDir(filepath.Join(testdata, "dest"))
	require.NoError(t, err)
	for _, file := range oldDestFiles {
		if file.Type().IsRegular() {
			content, err := os.ReadFile(filepath.Join(testdata, "dest", file.Name()))
			require.NoError(t, err)
			err = os.WriteFile(filepath.Join(newDest, file.Name()), content, 0644)
			require.NoError(t, err)
		}
	}

	options := updatetypes.Options{
		OriginPath:  filepath.Join(testdata, "origin"),
		UpdatedPath: filepath.Join(testdata, "updated"),
		LocalPath:   newDest,
	}
	err = updater.Update(options)
	require.NoError(t, err)

	destBytes, err := os.ReadFile(filepath.Join(newDest, "configmap.yaml"))
	require.NoError(t, err)
	expBytes, err := os.ReadFile(filepath.Join(testdata, "expected", "configmap.yaml"))
	require.NoError(t, err)

	if diff := cmp.Diff(expBytes, destBytes); diff != "" {
		t.Errorf("unexpected result (-want, +got): %s", diff)
	}
}
