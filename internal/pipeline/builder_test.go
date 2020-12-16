// Copyright 2020 Google LLC
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
package pipeline_test

import (
	"testing"

	. "github.com/GoogleContainerTools/kpt/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestNewBuilder(t *testing.T) {
	b := NewBuilder()
	if !assert.EqualValues(t, b.Build(), DefaultPipeline()) {
		t.FailNow()
	}
}

var expected Pipeline = Pipeline{
	Sources: []string{
		"a",
		"b",
	},
	Generators: []Function{
		{
			Image: "gcr.io/kpt-functions/generate-folders",
			Config: *yaml.MustParse(`apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
name: root-hierarchy
namespace: hierarchy # {"$kpt-set":"namespace"}`).YNode(),
		},
	},
	Transformers: []Function{
		{
			Image:      "patch-strategic-merge",
			ConfigPath: "./patch.yaml",
		},
		{
			Image: "gcr.io/kpt-functions/set-annotation",
			ConfigMap: map[string]string{
				"environment": "dev",
			},
		},
	},
	Validators: []Function{
		{
			Image: "gcr.io/kpt-functions/policy-controller-validate",
		},
	},
}

func TestAdd(t *testing.T) {
	actual := NewBuilderWithPipeline(&Pipeline{}).
		AddSources("a", "b").
		AddGenerators(expected.Generators...).
		AddTransformers(expected.Transformers...).
		AddValidators(expected.Validators...).
		Build()

	if !isPipelineEqual(t, *actual, expected) {
		t.Fatalf("build result is different from expected")
	}
}

func TestSet(t *testing.T) {
	actual := NewBuilder().
		SetName("").
		SetKind("").
		SetAPIVersion("").
		SetSources([]string{"a", "b"}).
		SetGenerators(expected.Generators).
		SetTransformers(expected.Transformers).
		SetValidators(expected.Validators).
		Build()
	if !isPipelineEqual(t, *actual, expected) {
		t.Fatalf("build result is different from expected")
	}
}
