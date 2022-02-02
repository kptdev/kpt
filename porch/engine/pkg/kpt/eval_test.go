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

package kpt

import (
	"context"
	"strings"
	"testing"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetLabels(t *testing.T) {
	k := &runner{}

	const pkgYaml = `# Comment
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
    name: blueprints-project-bucket
    namespace: config-control
spec:
    storageClass: standard
`

	filter, err := k.NewRunner(context.Background(), &v1.Function{
		Image: "gcr.io/kpt-fn/set-labels:v0.1.5",
		ConfigMap: map[string]string{
			"label-key": "label-value",
		},
	}, fn.RunnerOptions{})

	if err != nil {
		t.Errorf("Eval failed: %v", err)
	}

	var result []*yaml.RNode
	var writer kio.WriterFunc = func(r []*yaml.RNode) error {
		result = r
		return nil
	}

	p := &kio.Pipeline{
		Inputs:                []kio.Reader{&kio.ByteReader{Reader: strings.NewReader(pkgYaml)}},
		Filters:               []kio.Filter{filter},
		Outputs:               []kio.Writer{writer},
		ContinueOnEmptyResult: false,
	}

	if err := p.Execute(); err != nil {
		t.Errorf("Failed to evaluate function: %v", err)
	}

	if got, want := len(result), 1; got != want {
		t.Errorf("Expected single resource in the result. got %d", got)
	}
	for _, n := range result {
		labels := n.GetLabels()
		if got, ok := labels["label-key"]; !ok {
			t.Error("label 'label-key' was not set")
		} else if want := "label-value"; got != want {
			t.Errorf("unexpected label-key value; got %q, want %q", got, want)
		}
	}
}
