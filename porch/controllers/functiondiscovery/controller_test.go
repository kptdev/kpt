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

package functiondiscovery

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	api "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/test/testhelpers"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"sigs.k8s.io/yaml"
)

func TestFunctionDiscoveryController(t *testing.T) {
	h := testhelpers.NewHarness(t)

	oci := h.StartOCIServer()
	ociURL := oci.Endpoint() + "/testrepo"

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
    registry: "{{ociURL}}"
`

	// Push some empty functions that we expect to be discovered
	imageTags := []string{
		"enable-gcp-services:v0",
		"enable-gcp-services:v0.0",
		"enable-gcp-services:v0.0.0",
		"set-labels:unstable",
		"set-labels:v0",
		"set-labels:v0.2",
		"set-labels:v0.2.0",
		"set-labels:v0.1",
		"set-labels:v0.1.5",
		"set-labels:v0.1.3",
		"set-labels:v0.1.4",
	}

	for _, image := range imageTags {
		img := mutate.Annotations(empty.Image, map[string]string{
			"dev.kpt.fn.meta.description":      "description for " + image,
			"dev.kpt.fn.meta.documentationurl": "documentation for " + image,
		}).(v1.Image)
		dest := ociURL + "/" + image
		if err := crane.Push(img, dest); err != nil {
			t.Fatalf("failed to push image %q: %v", dest, err)
		}
	}

	repoYAML = strings.ReplaceAll(repoYAML, "{{ociURL}}", ociURL)

	repo := &api.Repository{}
	if err := yaml.Unmarshal([]byte(repoYAML), repo); err != nil {
		t.Fatalf("error parsing yaml: %v", err)
	}

	r := &FunctionReconciler{}
	objectsToApply, err := r.buildObjectsToApply(h.Ctx, repo)
	if err != nil {
		t.Fatalf("error from buildObjectsToApply: %v", err)
	}

	sort.Slice(objectsToApply, func(i, j int) bool {
		return objectsToApply[i].GetName() < objectsToApply[j].GetName()
	})

	got := testhelpers.ToYAML(h, objectsToApply)
	got = strings.ReplaceAll(got, ociURL+"/", "ociurl/")

	h.AssertMatchesFile(filepath.Join(testdir, "expected.yaml"), got)
}
