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

package engine

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func TestRender(t *testing.T) {
	render := &renderPackageMutation{
		renderer: kpt.NewPlaceholderRenderer(),
		runtime:  kpt.NewPlaceholderFunctionRuntime(),
	}

	const path = "bucket.yaml"
	const annotation = "porch.kpt.dev/rendered"
	const content = `# Comment
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
    name: blueprints-project-bucket
    namespace: config-control
spec:
    storageClass: standard
`

	resources := repository.PackageResources{
		Contents: map[string]string{
			path: content,
		},
	}

	output, _, err := render.Apply(context.Background(), resources)
	if err != nil {
		t.Errorf("package render failed: %v", err)
	}

	if got, want := len(output.Contents), 1; got != want {
		t.Errorf("Expected single resource in the result. got %d", got)
	}

	result, err := kio.ParseAll(output.Contents[path])
	if err != nil {
		t.Errorf("Failed to parse rendered package content: %v", err)
	}

	for _, n := range result {
		annotations := n.GetAnnotations()
		if got, ok := annotations[annotation]; !ok {
			t.Errorf("expected %q annotation, got none", annotation)
		} else if want := "yes"; got != want {
			t.Errorf("%q annotation: got %q, want %q", annotation, got, want)
		}
	}
}
