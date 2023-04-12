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

package kpt

import (
	"bytes"
	"context"
	"path"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func ReadResourceList(t *testing.T, pkgdir string, config *yaml.RNode) []byte {
	r := &kio.LocalPackageReader{
		PackagePath:     pkgdir,
		WrapBareSeqNode: true,
	}

	var rl bytes.Buffer
	w := &kio.ByteWriter{
		Writer:                &rl,
		KeepReaderAnnotations: true,
		WrappingKind:          kio.ResourceListKind,
		WrappingAPIVersion:    kio.ResourceListAPIVersion,
		FunctionConfig:        config,
	}
	if err := (kio.Pipeline{Inputs: []kio.Reader{r}, Outputs: []kio.Writer{w}}).Execute(); err != nil {
		t.Fatalf("Failed to load package %q", pkgdir)
		return nil
	} else {
		return rl.Bytes()
	}
}

func TestSetLabels(t *testing.T) {
	r := &runtime{}
	runner, err := r.GetRunner(context.Background(), &v1.Function{
		Image: "gcr.io/kpt-fn/set-labels:v0.1.5",
	})
	if err != nil {
		t.Errorf("GetRunner failed: %v", err)
	}

	config, err := fnruntime.NewConfigMap(map[string]string{
		"label-key": "label-value",
	})
	if err != nil {
		t.Fatalf("Failed to create function config map: %v", err)
	}

	input := ReadResourceList(t, path.Join(".", "testdata", "bucket"), config)
	var output bytes.Buffer
	if err := runner.Run(bytes.NewReader(input), &output); err != nil {
		t.Errorf("Eval failed: %v", err)
	}

	t.Log(output.String())

	reader := kio.ByteReader{Reader: &output}
	result, err := reader.Read()
	if err != nil {
		t.Errorf("Reading results failed")
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
