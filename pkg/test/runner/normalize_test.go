// Copyright 2026 The kpt Authors
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

package runner

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNormalizeDiff_KptfileMapOrderInsensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,4 +1,5 @@
-  message: |
-  reason: RenderFailed
+reason: RenderFailed
+  message: |
 status: "False"`

	expected := `diff --git a/Kptfile b/Kptfile
index fedcba9..7654321 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,4 +1,5 @@
-reason: RenderFailed
-message: |
+message: |
+reason: RenderFailed
 status: "False"`

	gotActual, err := normalizeDiff(actual, "")
	if err != nil {
		t.Fatalf("normalizeDiff(actual) failed: %v", err)
	}

	gotExpected, err := normalizeDiff(expected, "")
	if err != nil {
		t.Fatalf("normalizeDiff(expected) failed: %v", err)
	}

	if diff := cmp.Diff(gotExpected, gotActual); diff != "" {
		t.Fatalf("normalized diffs mismatch (-want, +got): %s", diff)
	}
}

func TestNormalizeDiff_NonMapRunPreservesOrder(t *testing.T) {
	input := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,2 +1,2 @@
- kind: Kptfile
+ kind: Kptfile
 context-line`
	want := `diff --git a/Kptfile b/Kptfile
index NORMALIZED 100644
--- a/Kptfile
+++ b/Kptfile
@@ NORMALIZED @@
-kind: Kptfile
+kind: Kptfile
 context-line`

	got, err := normalizeDiff(input, "")
	if err != nil {
		t.Fatalf("normalizeDiff failed: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected normalization for non-map run (-want, +got): %s", diff)
	}
}

func TestNormalizeDiff_NonKRMReasonMessageOrderInsensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 1111111..2222222 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,4 +1,9 @@
+status:
+  conditions:
+  - type: Rendered
+    status: "False"
+    message: render failed
+    reason: RenderFailed`

	expected := `diff --git a/Kptfile b/Kptfile
index aaaaaaa..bbbbbbb 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,4 +1,9 @@
+status:
+  conditions:
+  - type: Rendered
+    status: "False"
+    reason: RenderFailed
+    message: render failed`

	gotActual, err := normalizeDiff(actual, "")
	if err != nil {
		t.Fatalf("normalizeDiff(actual) failed: %v", err)
	}
	gotExpected, err := normalizeDiff(expected, "")
	if err != nil {
		t.Fatalf("normalizeDiff(expected) failed: %v", err)
	}

	if diff := cmp.Diff(gotExpected, gotActual); diff != "" {
		t.Fatalf("normalized diffs mismatch (-want, +got): %s", diff)
	}
}

func TestNormalizeDiff_MutationStepFieldOrderInsensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 3333333..4444444 100644
--- a/Kptfile
+++ b/Kptfile
@@ -10,6 +10,12 @@
+  renderStatus:
+    mutationSteps:
+      - image: fn:set-namespace
+        exitCode: 0
+        results:
+          - message: ok`

	expected := `diff --git a/Kptfile b/Kptfile
index ccccccc..ddddddd 100644
--- a/Kptfile
+++ b/Kptfile
@@ -10,6 +10,12 @@
+  renderStatus:
+    mutationSteps:
+      - image: fn:set-namespace
+        results:
+          - message: ok
+        exitCode: 0`

	gotActual, err := normalizeDiff(actual, "")
	if err != nil {
		t.Fatalf("normalizeDiff(actual) failed: %v", err)
	}
	gotExpected, err := normalizeDiff(expected, "")
	if err != nil {
		t.Fatalf("normalizeDiff(expected) failed: %v", err)
	}

	if diff := cmp.Diff(gotExpected, gotActual); diff != "" {
		t.Fatalf("normalized diffs mismatch (-want, +got): %s", diff)
	}
}
