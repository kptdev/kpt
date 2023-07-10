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

package engine

import (
	"context"
	"strings"
	"testing"

	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/engine/fake"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/google/go-cmp/cmp"
)

func TestSomething(t *testing.T) {
	testCases := map[string]struct {
		repoPkgRev   repository.PackageRevision
		newApiPkgRev *api.PackageRevision
		hasPatch     bool
		patch        api.PatchSpec
	}{
		"no gates or conditions": {
			repoPkgRev: &fake.PackageRevision{
				Kptfile: kptfile.KptFile{},
			},
			newApiPkgRev: &api.PackageRevision{
				Spec: api.PackageRevisionSpec{},
			},
			hasPatch: false,
		},
		"first gate and condition added": {
			repoPkgRev: &fake.PackageRevision{
				Kptfile: kptfile.KptFile{},
			},
			newApiPkgRev: &api.PackageRevision{
				Spec: api.PackageRevisionSpec{
					ReadinessGates: []api.ReadinessGate{
						{
							ConditionType: "foo",
						},
					},
				},
				Status: api.PackageRevisionStatus{
					Conditions: []api.Condition{
						{
							Type:   "foo",
							Status: api.ConditionTrue,
						},
					},
				},
			},
			hasPatch: true,
			patch: api.PatchSpec{
				File: kptfile.KptFileName,
				Contents: strings.TrimSpace(`
--- Kptfile
+++ Kptfile
@@ -1 +1,7 @@
-{}
+info:
+  readinessGates:
+  - conditionType: foo
+status:
+  conditions:
+  - type: foo
+    status: "True"				
`) + "\n",
				PatchType: api.PatchTypePatchFile,
			},
		},
		"additional readinessGates and conditions added": {
			repoPkgRev: &fake.PackageRevision{
				Kptfile: kptfile.KptFile{
					Info: &kptfile.PackageInfo{
						ReadinessGates: []kptfile.ReadinessGate{
							{
								ConditionType: "foo",
							},
						},
					},
					Status: &kptfile.Status{
						Conditions: []kptfile.Condition{
							{
								Type:   "foo",
								Status: kptfile.ConditionTrue,
							},
						},
					},
				},
			},
			newApiPkgRev: &api.PackageRevision{
				Spec: api.PackageRevisionSpec{
					ReadinessGates: []api.ReadinessGate{
						{
							ConditionType: "foo",
						},
						{
							ConditionType: "bar",
						},
					},
				},
				Status: api.PackageRevisionStatus{
					Conditions: []api.Condition{
						{
							Type:    "foo",
							Status:  api.ConditionTrue,
							Reason:  "reason",
							Message: "message",
						},
						{
							Type:    "bar",
							Status:  api.ConditionFalse,
							Reason:  "reason",
							Message: "message",
						},
					},
				},
			},
			hasPatch: true,
			patch: api.PatchSpec{
				File: kptfile.KptFileName,
				Contents: strings.TrimSpace(`
--- Kptfile
+++ Kptfile
@@ -1,7 +1,14 @@
 info:
   readinessGates:
   - conditionType: foo
+  - conditionType: bar
 status:
   conditions:
   - type: foo
     status: "True"
+    reason: reason
+    message: message
+  - type: bar
+    status: "False"
+    reason: reason
+    message: message
`) + "\n",
				PatchType: api.PatchTypePatchFile,
			},
		},
		"no changes": {
			repoPkgRev: &fake.PackageRevision{
				Kptfile: kptfile.KptFile{
					Info: &kptfile.PackageInfo{
						ReadinessGates: []kptfile.ReadinessGate{
							{
								ConditionType: "foo",
							},
							{
								ConditionType: "bar",
							},
						},
					},
					Status: &kptfile.Status{
						Conditions: []kptfile.Condition{
							{
								Type:    "foo",
								Status:  kptfile.ConditionTrue,
								Reason:  "reason",
								Message: "message",
							},
							{
								Type:    "bar",
								Status:  kptfile.ConditionFalse,
								Reason:  "reason",
								Message: "message",
							},
						},
					},
				},
			},
			newApiPkgRev: &api.PackageRevision{
				Spec: api.PackageRevisionSpec{
					ReadinessGates: []api.ReadinessGate{
						{
							ConditionType: "foo",
						},
						{
							ConditionType: "bar",
						},
					},
				},
				Status: api.PackageRevisionStatus{
					Conditions: []api.Condition{
						{
							Type:    "foo",
							Status:  api.ConditionTrue,
							Reason:  "reason",
							Message: "message",
						},
						{
							Type:    "bar",
							Status:  api.ConditionFalse,
							Reason:  "reason",
							Message: "message",
						},
					},
				},
			},
			hasPatch: false,
		},
		"readinessGates and conditions removed": {
			repoPkgRev: &fake.PackageRevision{
				Kptfile: kptfile.KptFile{
					Info: &kptfile.PackageInfo{
						ReadinessGates: []kptfile.ReadinessGate{
							{
								ConditionType: "foo",
							},
							{
								ConditionType: "bar",
							},
						},
					},
					Status: &kptfile.Status{
						Conditions: []kptfile.Condition{
							{
								Type:    "foo",
								Status:  kptfile.ConditionTrue,
								Reason:  "reason",
								Message: "message",
							},
							{
								Type:    "bar",
								Status:  kptfile.ConditionFalse,
								Reason:  "reason",
								Message: "message",
							},
						},
					},
				},
			},
			newApiPkgRev: &api.PackageRevision{
				Spec: api.PackageRevisionSpec{
					ReadinessGates: []api.ReadinessGate{
						{
							ConditionType: "foo",
						},
					},
				},
				Status: api.PackageRevisionStatus{
					Conditions: []api.Condition{
						{
							Type:   "foo",
							Status: api.ConditionTrue,
						},
					},
				},
			},
			hasPatch: true,
			patch: api.PatchSpec{
				File: kptfile.KptFileName,
				Contents: strings.TrimSpace(`
--- Kptfile
+++ Kptfile
@@ -1,14 +1,7 @@
 info:
   readinessGates:
   - conditionType: foo
-  - conditionType: bar
 status:
   conditions:
   - type: foo
     status: "True"
-    reason: reason
-    message: message
-  - type: bar
-    status: "False"
-    reason: reason
-    message: message
`) + "\n",
				PatchType: api.PatchTypePatchFile,
			},
		},
	}

	for tn := range testCases {
		tc := testCases[tn]
		t.Run(tn, func(t *testing.T) {
			task, hasPatch, err := createKptfilePatchTask(context.Background(), tc.repoPkgRev, tc.newApiPkgRev)
			if err != nil {
				t.Fatal(err)
			}

			if tc.hasPatch && !hasPatch {
				t.Errorf("expected patch, but didn't get one")
			}
			if !tc.hasPatch {
				if hasPatch {
					t.Errorf("expected no patch, but got one")
				}
				return
			}

			if diff := cmp.Diff(tc.patch, task.Patch.Patches[0]); diff != "" {
				t.Errorf("Unexpected result (-want, +got): %s", diff)
			}
		})
	}
}
