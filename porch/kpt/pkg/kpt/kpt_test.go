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
	"strings"
	"testing"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetLabels(t *testing.T) {
	k := &evaluator{}

	const path = "bucket.yaml"
	const pathAnnotation = "internal.config.kubernetes.io/package-path"
	const pkgYaml = `# Comment
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
    name: blueprints-project-bucket
    namespace: config-control
spec:
    storageClass: standard
`
	const cfgYaml = `
apiVersion: v1
kind: ConfigMap
metadata:
    name: config
data:
    label-key: label-value
`

	pkg := &kio.ByteReader{
		Reader: strings.NewReader(pkgYaml),
		SetAnnotations: map[string]string{
			pathAnnotation: path,
		},
	}
	cfg := &kio.ByteReader{Reader: strings.NewReader(cfgYaml)}

	var result []*yaml.RNode = nil
	var output kio.WriterFunc = func(o []*yaml.RNode) error { result = o; return nil }

	if err := k.OldEval(pkg, "gcr.io/kpt-fn/set-labels:v0.1.5", cfg, output); err != nil {
		t.Errorf("function eval failed: %v", err)
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

		annotations := n.GetAnnotations()
		if got, ok := annotations[pathAnnotation]; !ok {
			t.Errorf("expected %q annotation, got none", pathAnnotation)
		} else if want := path; got != want {
			t.Errorf("%q annotation: got %q, want %q", pathAnnotation, got, want)
		}
	}
}
