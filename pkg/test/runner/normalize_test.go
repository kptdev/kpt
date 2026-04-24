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
	"strings"
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
+kind: Kptfile`

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

func TestNormalizeDiff_KptfileMultilineAndTabOrderInsensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,5 +1,8 @@
-message: |-
-  pkg.render: pkg .:
+reason: RenderFailed
+	pipeline.run: pkg ./subpkg: already handled error
+message: |-
 status: "False"`

	expected := `diff --git a/Kptfile b/Kptfile
index fedcba9..7654321 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,5 +1,8 @@
-  pkg.render: pkg .:
-message: |-
+message: |-
+pipeline.run: pkg ./subpkg: already handled error
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

func TestNormalizeDiff_KptfileQuotedScalarInsensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,3 +1,3 @@
-level: "root"
+level: root`

	expected := `diff --git a/Kptfile b/Kptfile
index fedcba9..7654321 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,3 +1,3 @@
-level: root
+level: "root"`

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

func TestNormalizeDiff_HunkHeaderContextInsensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,4 +1,8 @@ metadata:
+status: True`

	expected := `diff --git a/Kptfile b/Kptfile
index fedcba9..7654321 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,4 +1,8 @@ pipeline:
+status: True`

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

func TestNormalizeDiff_KptfileQuotedScalarWithSpacesInsensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,3 +1,3 @@
-exec: "sed -e 's/foo/bar/'"
+exec: sed -e 's/foo/bar/'`

	expected := `diff --git a/Kptfile b/Kptfile
index fedcba9..7654321 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,3 +1,3 @@
-exec: sed -e 's/foo/bar/'
+exec: "sed -e 's/foo/bar/'"`

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

func TestNormalizeDiff_KptfileSingleQuotedScalarInsensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,3 +1,3 @@
-stderr: 'failed to evaluate function: error: function failure'
+stderr: failed to evaluate function: error: function failure`

	expected := `diff --git a/Kptfile b/Kptfile
index fedcba9..7654321 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,3 +1,3 @@
-stderr: failed to evaluate function: error: function failure
+stderr: 'failed to evaluate function: error: function failure'`

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

func TestNormalizeDiff_KptfileEscapedSingleQuotedScalarInsensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,3 +1,3 @@
-errorSummary: 'can''t render package'
+errorSummary: can't render package`

	expected := `diff --git a/Kptfile b/Kptfile
index fedcba9..7654321 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,3 +1,3 @@
-errorSummary: can't render package
+errorSummary: 'can''t render package'`

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

func TestNormalizeDiff_KptfileContextLineInsensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,4 +1,8 @@
 metadata:
   labels:
-tier: backend
+tier: backend
 pipeline:
   mutators:`

	expected := `diff --git a/Kptfile b/Kptfile
index fedcba9..7654321 100644
--- a/Kptfile
+++ b/Kptfile
@@ -7,4 +11,8 @@
 pipeline:
   mutators:
-tier: backend
+tier: backend`

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

func TestNormalizeDiff_NoNewlineMarkerInsensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,3 +1,3 @@
-stderr: failed to evaluate function: error: function failure
\ No newline at end of file
+stderr: failed to evaluate function: error: function failure`

	expected := `diff --git a/Kptfile b/Kptfile
index fedcba9..7654321 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,3 +1,3 @@
-stderr: failed to evaluate function: error: function failure
+stderr: failed to evaluate function: error: function failure
\ No newline at end of file`

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

func TestNormalizeDiff_KptfileListOrderSensitive(t *testing.T) {
	actual := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,5 +1,5 @@
-  - image: fn:first
-  - image: fn:second
+  - image: fn:second
+  - image: fn:first`

	expected := `diff --git a/Kptfile b/Kptfile
index fedcba9..7654321 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,5 +1,5 @@
-  - image: fn:first
-  - image: fn:second
+  - image: fn:first
+  - image: fn:second`

	gotActual, err := normalizeDiff(actual, "")
	if err != nil {
		t.Fatalf("normalizeDiff(actual) failed: %v", err)
	}
	gotExpected, err := normalizeDiff(expected, "")
	if err != nil {
		t.Fatalf("normalizeDiff(expected) failed: %v", err)
	}

	if diff := cmp.Diff(gotExpected, gotActual); diff == "" {
		t.Fatalf("list item reorder should remain detectable after normalization")
	}
}

func TestNormalizeDiff_SubdirKptfileHeaderDetected(t *testing.T) {
	actual := `diff --git a/subpkg/Kptfile b/subpkg/Kptfile
index 1234567..89abcde 100644
--- a/subpkg/Kptfile
+++ b/subpkg/Kptfile
@@ -1,4 +1,4 @@
-reason: RenderFailed
-message: render failed
+message: render failed
+reason: RenderFailed`

	expected := `diff --git a/subpkg/Kptfile b/subpkg/Kptfile
index fedcba9..7654321 100644
--- a/subpkg/Kptfile
+++ b/subpkg/Kptfile
@@ -1,4 +1,4 @@
-message: render failed
-reason: RenderFailed
+reason: RenderFailed
+message: render failed`

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

func TestNormalizeDiff_NonKptfileIndentDriftInsensitive(t *testing.T) {
	// Goldens committed in this branch stripped YAML indentation from
	// non-Kptfile hunks, while the actual fn output preserves 2-space
	// indent. Both sides should normalize to the same form.
	actual := `diff --git a/resources.yaml b/resources.yaml
index 1234567..89abcde 100644
--- a/resources.yaml
+++ b/resources.yaml
@@ -15,6 +15,7 @@ apiVersion: apps/v1
 kind: Deployment
 metadata:
   name: nginx-deployment
+  namespace: staging
 spec:
   replicas: 3`

	expected := `diff --git a/resources.yaml b/resources.yaml
index fedcba9..7654321 100644
--- a/resources.yaml
+++ b/resources.yaml
@@ -15,6 +15,7 @@ apiVersion: apps/v1
 kind: Deployment
 metadata:
 name: nginx-deployment
+namespace: staging
 spec:
 replicas: 3`

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

func TestNormalizeDiff_NonKptfilePreservesContextLines(t *testing.T) {
	// Unlike Kptfile diffs (where context lines are unstable anchors
	// and dropped), non-Kptfile diffs keep context lines — they just
	// get their leading whitespace stripped.
	input := `diff --git a/resources.yaml b/resources.yaml
index 1234567..89abcde 100644
--- a/resources.yaml
+++ b/resources.yaml
@@ -1,4 +1,5 @@
 kind: Deployment
 metadata:
   name: nginx-deployment
+  namespace: staging
 spec:`

	got, err := normalizeDiff(input, "")
	if err != nil {
		t.Fatalf("normalizeDiff failed: %v", err)
	}

	for _, want := range []string{" kind: Deployment", " metadata:", " name: nginx-deployment", "+namespace: staging", " spec:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected normalized output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestNormalizeDiff_KptfilePreservesNestedStructure(t *testing.T) {
	input := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,3 +1,12 @@
+status:
+  renderStatus:
+    mutationSteps:
+      - image: ghcr.io/kptdev/krm-functions-catalog/wasm/set-namespace:v0.5.1
+        configMap:
+          namespace: staging
+      - image: ghcr.io/kptdev/krm-functions-catalog/wasm/set-labels:v0.2.4
+        configMap:
+          tier: backend`

	got, err := normalizeDiff(input, "")
	if err != nil {
		t.Fatalf("normalizeDiff failed: %v", err)
	}

	if !strings.Contains(got, "+- image: ghcr.io/kptdev/krm-functions-catalog/wasm/set-namespace:v0.5.1\n+configMap:\n+namespace: staging") {
		t.Fatalf("expected configMap lines to stay grouped with set-namespace image line, got:\n%s", got)
	}
}

// TestNormalizeDiff_StripREAppliesAfterKptfileIndentStrip pins the contract
// that per-test diffStripRegEx patterns are evaluated against the
// post-indent-strip form of Kptfile +/- lines. Test configs (e.g.
// image-pull-policy-never, missing-fn-image) therefore use `\s*` rather
// than `\s+` when matching keys like `stderr:` so they survive the
// Kptfile indent-stripping step. This test locks that behavior in so
// future normalizer changes can't silently re-break the interaction.
func TestNormalizeDiff_StripREAppliesAfterKptfileIndentStrip(t *testing.T) {
	// Raw pre-normalization diff, as produced by git diff against a Kptfile
	// where the rendered pipeline wrote a multi-line stderr block.
	input := `diff --git a/Kptfile b/Kptfile
index 1234567..89abcde 100644
--- a/Kptfile
+++ b/Kptfile
@@ -1,4 +1,8 @@
+status:
+  renderStatus:
+    mutationSteps:
+      - image: ghcr.io/kptdev/krm-functions-catalog/dne:latest
+        exitCode: 125
+        stderr: |-
+          Error: ghcr.io/kptdev/krm-functions-catalog/dne:latest: image not known`

	// \s* (not \s+) is the key: after the normalizer strips the leading
	// whitespace off Kptfile +/- lines, the regex still has to match
	// `+stderr:` with zero spaces after the `+`.
	stripRE := `\+\s*stderr:|Error:.*image not known`

	got, err := normalizeDiff(input, stripRE)
	if err != nil {
		t.Fatalf("normalizeDiff failed: %v", err)
	}

	if strings.Contains(got, "stderr:") {
		t.Fatalf("expected stderr label to be stripped, got:\n%s", got)
	}
	if strings.Contains(got, "image not known") {
		t.Fatalf("expected stderr body (matching Error:.*image not known) to be stripped, got:\n%s", got)
	}

	// Sanity: the relaxed \s* also matches the pre-indent-strip form, so a
	// stripRE authored for the raw diff still wins after normalization.
	stripREStrictPlus := `\+\s+stderr:` // old form with \s+
	gotStrict, err := normalizeDiff(input, stripREStrictPlus)
	if err != nil {
		t.Fatalf("normalizeDiff failed: %v", err)
	}
	if !strings.Contains(gotStrict, "stderr:") {
		t.Fatalf("regression-guard: the tight \\s+ pattern *should* fail to strip the post-indent-strip line (that's the bug we fixed); got:\n%s", gotStrict)
	}
}
