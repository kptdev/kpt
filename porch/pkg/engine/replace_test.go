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
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestReplaceResources(t *testing.T) {
	ctx := context.Background()

	input := readPackage(t, filepath.Join("testdata", "replace"))
	nocomment := removeComments(t, input)

	replace := &mutationReplaceResources{
		newResources: &v1alpha1.PackageRevisionResources{
			Spec: v1alpha1.PackageRevisionResourcesSpec{
				Resources: nocomment.Contents,
			},
		},
		oldResources: &v1alpha1.PackageRevisionResources{
			Spec: v1alpha1.PackageRevisionResourcesSpec{
				Resources: input.Contents,
			},
		},
	}

	output, _, err := replace.Apply(ctx, input)
	if err != nil {
		t.Fatalf("mutationReplaceResources.Apply failed: %v", err)
	}

	if !cmp.Equal(input, output) {
		t.Errorf("Diff: (-want,+got): %s", cmp.Diff(input, output))
	}
}

func removeComments(t *testing.T, r repository.PackageResources) repository.PackageResources {
	t.Helper()

	out := repository.PackageResources{
		Contents: map[string]string{},
	}

	for k, v := range r.Contents {
		var data interface{}
		if err := yaml.Unmarshal([]byte(v), &data); err != nil {
			t.Fatalf("Failed to unmarshal %q: %v", k, err)
		}

		var nocomment bytes.Buffer
		encoder := yaml.NewEncoder(&nocomment)
		encoder.SetIndent(0)
		if err := encoder.Encode(data); err != nil {
			t.Fatalf("Failed to re-encode yaml output: %v", err)
		}

		out.Contents[k] = nocomment.String()
	}

	return out
}
