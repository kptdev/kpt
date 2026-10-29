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

// TestUpdate_NonKRMFileNotMergedByContent verifies a .yaml file whose
// content isn't valid KRM is raw-copied/diffed, never merged.
func TestUpdate_NonKRMFileNotMergedByContent(t *testing.T) {
	origin := pkgbuilder.NewRootPkg().
		WithResource(pkgbuilder.DeploymentResource).
		WithFile("values.yaml", "replicaCount: 3\nimage:\n  repository: nginx\n  tag: latest\n")
	local := pkgbuilder.NewRootPkg().
		WithKptfile(
			pkgbuilder.NewKptfile().
				WithUpstream(kptRepo, "/", "master", "resource-merge").
				WithUpstreamLock(kptRepo, "/", "master", "abc123"),
		).
		WithResource(pkgbuilder.DeploymentResource).
		WithFile("values.yaml", "replicaCount: 3\nimage:\n  repository: nginx\n  tag: latest\n# local edit\n")
	updated := pkgbuilder.NewRootPkg().
		WithResource(pkgbuilder.DeploymentResource).
		WithFile("values.yaml", "replicaCount: 5\ntotallyDifferentField: yes\n")
	// Locally modified: local's edit is kept untouched, same as
	// ReplaceNonKRMFiles already does for any other non-KRM file.
	expected := pkgbuilder.NewRootPkg().
		WithKptfile(
			pkgbuilder.NewKptfile().
				WithUpstream(kptRepo, "/", "master", "resource-merge").
				WithUpstreamLock(kptRepo, "/", "master", "abc123"),
		).
		WithResource(pkgbuilder.DeploymentResource).
		WithFile("values.yaml", "replicaCount: 3\nimage:\n  repository: nginx\n  tag: latest\n# local edit\n")

	repos := testutil.EmptyReposInfo
	originPath := origin.ExpandPkg(t, repos)
	localPath := local.ExpandPkg(t, repos)
	updatedPath := updated.ExpandPkg(t, repos)

	updater := &update.ResourceMergeUpdater{}
	err := updater.Update(updatetypes.Options{
		RelPackagePath: "/",
		OriginPath:     originPath,
		LocalPath:      localPath,
		UpdatedPath:    updatedPath,
		IsRoot:         true,
	})
	require.NoError(t, err)

	gotContent, err := os.ReadFile(filepath.Join(localPath, "values.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "replicaCount: 3\nimage:\n  repository: nginx\n  tag: latest\n# local edit\n", string(gotContent),
		"non-KRM file must never be field-merged, and a locally modified file is kept, not overwritten")

	expectedPath := expected.ExpandPkg(t, repos)
	testutil.KptfileAwarePkgEqual(t, localPath, expectedPath, false)
}

// TestUpdate_NonKRMFile_LocalOnlySurvives: a non-KRM file that exists
// only locally (absent from both origin and upstream) must survive an
// update untouched, not be deleted by the merge-destination writer
// before ReplaceNonKRMFiles runs.
func TestUpdate_NonKRMFile_LocalOnlySurvives(t *testing.T) {
	origin := pkgbuilder.NewRootPkg().
		WithResource(pkgbuilder.DeploymentResource)
	local := pkgbuilder.NewRootPkg().
		WithKptfile(
			pkgbuilder.NewKptfile().
				WithUpstream(kptRepo, "/", "master", "resource-merge").
				WithUpstreamLock(kptRepo, "/", "master", "abc123"),
		).
		WithResource(pkgbuilder.DeploymentResource).
		WithFile("values.yaml", "replicaCount: 3\n")
	updated := pkgbuilder.NewRootPkg().
		WithResource(pkgbuilder.DeploymentResource)

	repos := testutil.EmptyReposInfo
	originPath := origin.ExpandPkg(t, repos)
	localPath := local.ExpandPkg(t, repos)
	updatedPath := updated.ExpandPkg(t, repos)

	updater := &update.ResourceMergeUpdater{}
	err := updater.Update(updatetypes.Options{
		RelPackagePath: "/",
		OriginPath:     originPath,
		LocalPath:      localPath,
		UpdatedPath:    updatedPath,
		IsRoot:         true,
	})
	require.NoError(t, err)

	gotContent, err := os.ReadFile(filepath.Join(localPath, "values.yaml"))
	require.NoError(t, err, "local-only non-KRM file must not be deleted during update")
	assert.Equal(t, "replicaCount: 3\n", string(gotContent))
}

// TestUpdate_NonKRMListFileSurvives verifies a file with a bare `kind:
// List` (no apiVersion of its own) wrapping a valid inner resource is
// classified consistently by isKrmFile and the merge reader, so it's
// preserved rather than silently deleted as "no longer in the merge output."
func TestUpdate_NonKRMListFileSurvives(t *testing.T) {
	const listFileContent = `kind: List
items:
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: cm1
  data:
    key: value
`
	origin := pkgbuilder.NewRootPkg().
		WithResource(pkgbuilder.DeploymentResource).
		WithFile("list.yaml", listFileContent)
	local := pkgbuilder.NewRootPkg().
		WithKptfile(
			pkgbuilder.NewKptfile().
				WithUpstream(kptRepo, "/", "master", "resource-merge").
				WithUpstreamLock(kptRepo, "/", "master", "abc123"),
		).
		WithResource(pkgbuilder.DeploymentResource).
		WithFile("list.yaml", listFileContent)
	// Only the unrelated Deployment resource changes upstream; list.yaml
	// itself is untouched between origin and updated.
	updated := pkgbuilder.NewRootPkg().
		WithResource(pkgbuilder.ConfigMapResource).
		WithFile("list.yaml", listFileContent)

	repos := testutil.EmptyReposInfo
	originPath := origin.ExpandPkg(t, repos)
	localPath := local.ExpandPkg(t, repos)
	updatedPath := updated.ExpandPkg(t, repos)

	updater := &update.ResourceMergeUpdater{}
	err := updater.Update(updatetypes.Options{
		RelPackagePath: "/",
		OriginPath:     originPath,
		LocalPath:      localPath,
		UpdatedPath:    updatedPath,
		IsRoot:         true,
	})
	require.NoError(t, err)

	gotContent, err := os.ReadFile(filepath.Join(localPath, "list.yaml"))
	require.NoError(t, err, "list.yaml must not be deleted during update")
	assert.Equal(t, listFileContent, string(gotContent))
}

// TestUpdate_GenerateNameJobMerged verifies a Job using
// metadata.generateName (no metadata.name) is classified as KRM and
// goes through the real 3-way merge, so a local edit (added label) and
// an upstream edit (changed image) both land in the result. Before the
// fix, IsKRM required metadata.name unconditionally, so this file was
// misclassified non-KRM and routed to ReplaceNonKRMFiles's raw-copy
// path - which, for a locally modified file, keeps it untouched and
// silently drops the upstream image change entirely.
func TestUpdate_GenerateNameJobMerged(t *testing.T) {
	const origJob = `apiVersion: batch/v1
kind: Job
metadata:
  generateName: my-job-
spec:
  template:
    spec:
      containers:
      - name: c
        image: busybox:1.0
`
	const localJob = `apiVersion: batch/v1
kind: Job
metadata:
  generateName: my-job-
  labels:
    local: "true"
spec:
  template:
    spec:
      containers:
      - name: c
        image: busybox:1.0
`
	const updatedJob = `apiVersion: batch/v1
kind: Job
metadata:
  generateName: my-job-
spec:
  template:
    spec:
      containers:
      - name: c
        image: busybox:2.0
`
	origin := pkgbuilder.NewRootPkg().
		WithResource(pkgbuilder.DeploymentResource).
		WithFile("job.yaml", origJob)
	local := pkgbuilder.NewRootPkg().
		WithKptfile(
			pkgbuilder.NewKptfile().
				WithUpstream(kptRepo, "/", "master", "resource-merge").
				WithUpstreamLock(kptRepo, "/", "master", "abc123"),
		).
		WithResource(pkgbuilder.DeploymentResource).
		WithFile("job.yaml", localJob)
	updated := pkgbuilder.NewRootPkg().
		WithResource(pkgbuilder.DeploymentResource).
		WithFile("job.yaml", updatedJob)

	repos := testutil.EmptyReposInfo
	originPath := origin.ExpandPkg(t, repos)
	localPath := local.ExpandPkg(t, repos)
	updatedPath := updated.ExpandPkg(t, repos)

	updater := &update.ResourceMergeUpdater{}
	err := updater.Update(updatetypes.Options{
		RelPackagePath: "/",
		OriginPath:     originPath,
		LocalPath:      localPath,
		UpdatedPath:    updatedPath,
		IsRoot:         true,
	})
	require.NoError(t, err)

	gotContent, err := os.ReadFile(filepath.Join(localPath, "job.yaml"))
	require.NoError(t, err)
	got := string(gotContent)
	assert.Contains(t, got, "local: \"true\"", "the local label must survive the merge")
	assert.Contains(t, got, "busybox:2.0", "the upstream image change must not be silently dropped")
	assert.NotContains(t, got, "busybox:1.0", "the stale local image must be replaced, not kept verbatim")
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

// TestPreserveExplicitNull verifies a field explicitly nulled in dest is
// kept null (not dropped) when Options.PreserveNulls is set.
func TestPreserveExplicitNull(t *testing.T) {
	updater := update.ResourceMergeUpdater{}

	testdata, err := filepath.Abs(filepath.Join(".", "testdata", "keep-tagged-null-preserve"))
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
		OriginPath:    filepath.Join(testdata, "origin"),
		UpdatedPath:   filepath.Join(testdata, "updated"),
		LocalPath:     newDest,
		PreserveNulls: true,
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
