// Copyright 2021-2026 The kpt Authors
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

	"github.com/kptdev/kpt/internal/testutil"
	"github.com/kptdev/kpt/internal/testutil/pkgbuilder"
	"github.com/kptdev/kpt/internal/util/update"
	kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"
	"github.com/kptdev/kpt/pkg/lib/update/updatetypes"
	"github.com/stretchr/testify/assert"
)

const setLabelsImageV01 = "ghcr.io/kptdev/krm-functions-catalog/set-labels:v0.1"

func TestUpdate_FastForward(t *testing.T) {
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
						WithUpstream(kptRepo, "/", "master", "fast-forward").
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
						WithUpstream(kptRepo, "/", "master", "fast-forward").
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
								WithUpstream(kptRepo, "/", "main", "fast-forward"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream(kptRepo, "/", "main", "fast-forward"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			updated: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream(kptRepo, "/", "main", "fast-forward"),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream(kptRepo, "/", "main", "fast-forward"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
		},
		"Updates the Kptfile": {
			origin: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "main", "fast-forward"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			updated: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "v1.0", "fast-forward"),
				).
				WithResource(pkgbuilder.ConfigMapResource),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "v1.0", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.ConfigMapResource),
		},
		"render status on local Kptfile does not block fast-forward": {
			origin: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123").
						WithStatusCondition(kptfilev1.NewRenderedCondition(
							kptfilev1.ConditionTrue, kptfilev1.ReasonRenderSuccess, "")).
						WithStatusRenderStatus(
							[]kptfilev1.PipelineStepResult{{Image: setLabelsImageV01, ExitCode: 0}},
							nil, ""),
				).
				WithResource(pkgbuilder.DeploymentResource),
			updated: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource),
		},
		"non-rendered conditions are preserved after fast-forward": {
			origin: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithStatusCondition(kptfilev1.Condition{
							Type:   "Ready",
							Status: kptfilev1.ConditionTrue,
							Reason: "AllReady",
						}),
				).
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123").
						WithStatusCondition(kptfilev1.Condition{
							Type:   "Ready",
							Status: kptfilev1.ConditionTrue,
							Reason: "AllReady",
						}).
						WithStatusCondition(kptfilev1.NewRenderedCondition(
							kptfilev1.ConditionTrue, kptfilev1.ReasonRenderSuccess, "")).
						WithStatusRenderStatus(
							[]kptfilev1.PipelineStepResult{{Image: setLabelsImageV01, ExitCode: 0}},
							nil, ""),
				).
				WithResource(pkgbuilder.DeploymentResource),
			updated: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithStatusCondition(kptfilev1.Condition{
							Type:   "Ready",
							Status: kptfilev1.ConditionTrue,
							Reason: "AllReady",
						}),
				).
				WithResource(pkgbuilder.DeploymentResource),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123").
						WithStatusCondition(kptfilev1.Condition{
							Type:   "Ready",
							Status: kptfilev1.ConditionTrue,
							Reason: "AllReady",
						}),
				).
				WithResource(pkgbuilder.DeploymentResource),
		},
		"upstream render status is cleared after fast-forward": {
			origin: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			updated: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithStatusCondition(kptfilev1.NewRenderedCondition(
							kptfilev1.ConditionTrue, kptfilev1.ReasonRenderSuccess, "")).
						WithStatusRenderStatus(
							[]kptfilev1.PipelineStepResult{{Image: setLabelsImageV01, ExitCode: 0}},
							nil, ""),
				).
				WithResource(pkgbuilder.DeploymentResource),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource),
		},
		"failed render status is cleared after fast-forward": {
			origin: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource),
			local: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123").
						WithStatusCondition(kptfilev1.NewRenderedCondition(
							kptfilev1.ConditionFalse, kptfilev1.ReasonRenderFailed, "function failed")).
						WithStatusRenderStatus(
							[]kptfilev1.PipelineStepResult{{Image: setLabelsImageV01, ExitCode: 1, ExecutionError: "validation error"}},
							nil, "render failed"),
				).
				WithResource(pkgbuilder.DeploymentResource),
			updated: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource),
			relPackagePath: "/",
			isRoot:         true,
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstream(kptRepo, "/", "master", "fast-forward").
						WithUpstreamLock(kptRepo, "/", "master", "abc123"),
				).
				WithResource(pkgbuilder.DeploymentResource),
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			repos := testutil.EmptyReposInfo
			origin := tc.origin.ExpandPkg(t, repos)
			local := tc.local.ExpandPkg(t, repos)
			updated := tc.updated.ExpandPkg(t, repos)
			expected := tc.expected.ExpandPkg(t, repos)

			updater := &update.FastForwardUpdater{}

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

// TestFastForward_RenderStatusDoesNotMaskLocalEdits verifies that real local
// changes (e.g. pipeline edits) still block fast-forward even when render
// status is also present.
func TestFastForward_RenderStatusDoesNotMaskLocalEdits(t *testing.T) {
	repos := testutil.EmptyReposInfo

	origin := pkgbuilder.NewRootPkg().
		WithKptfile().
		WithResource(pkgbuilder.DeploymentResource).
		ExpandPkg(t, repos)

	local := pkgbuilder.NewRootPkg().
		WithKptfile(
			pkgbuilder.NewKptfile().
				WithUpstream(kptRepo, "/", "master", "fast-forward").
				WithUpstreamLock(kptRepo, "/", "master", "abc123").
				WithPipeline(pkgbuilder.NewFunction(setLabelsImageV01)).
				WithStatusCondition(kptfilev1.NewRenderedCondition(
					kptfilev1.ConditionTrue, kptfilev1.ReasonRenderSuccess, "")).
				WithStatusRenderStatus(
					[]kptfilev1.PipelineStepResult{{Image: setLabelsImageV01, ExitCode: 0}},
					nil, ""),
		).
		WithResource(pkgbuilder.DeploymentResource).
		ExpandPkg(t, repos)

	updated := pkgbuilder.NewRootPkg().
		WithKptfile().
		WithResource(pkgbuilder.DeploymentResource).
		ExpandPkg(t, repos)

	updater := &update.FastForwardUpdater{}

	err := updater.Update(updatetypes.Options{
		RelPackagePath: "/",
		OriginPath:     filepath.Join(origin, "/"),
		LocalPath:      filepath.Join(local, "/"),
		UpdatedPath:    filepath.Join(updated, "/"),
		IsRoot:         true,
	})
	if assert.Error(t, err, "local pipeline change should block fast-forward even with render status present") {
		assert.Contains(t, err.Error(), "local package files have been modified")
	}
}
