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

package functiondiscovery

import (
	"path/filepath"
	"testing"

	api "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/test/testhelpers"
	"sigs.k8s.io/yaml"
)

func TestFunctionDiscoveryController(t *testing.T) {
	h := testhelpers.NewHarness(t)

	testdir := filepath.Join("testdata", "testfunctiondiscoverycontroller", "oci")

	repoYAML := `
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: kpt-functions
  namespace: default
spec:
  description: Standard library of core kpt functions to manipulate KRM blueprints.
  content: Function
  type: oci
  oci:
    registry: gcr.io/kpt-fn
`

	repo := &api.Repository{}
	if err := yaml.Unmarshal([]byte(repoYAML), repo); err != nil {
		t.Fatalf("error parsing yaml: %v", err)
	}

	r := &FunctionReconciler{}
	objectsToApply, err := r.buildObjectsToApply(h.Ctx, repo)
	if err != nil {
		t.Fatalf("error from buildObjectsToApply: %v", err)
	}

	got := testhelpers.ToYAML(h, objectsToApply)

	h.AssertMatchesFile(filepath.Join(testdir, "expected.yaml"), got)
}
