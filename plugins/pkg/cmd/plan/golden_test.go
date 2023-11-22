// Copyright 2023 The kpt Authors
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

package plan

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kubebuilder-declarative-pattern/mockkubeapiserver"
	"sigs.k8s.io/yaml"
)

func TestPlanner(t *testing.T) {
	k8s, err := mockkubeapiserver.NewMockKubeAPIServer(":0")
	if err != nil {
		t.Fatalf("error building mock kube-apiserver: %v", err)
	}
	defer func() {
		if err := k8s.Stop(); err != nil {
			t.Fatalf("error closing mock kube-apiserver: %v", err)
		}
	}()
	addr, err := k8s.StartServing()
	if err != nil {
		t.Errorf("error starting mock kube-apiserver: %v", err)
	}
	klog.Infof("mock kubeapiserver will listen on %v", addr)

	restConfig := &rest.Config{
		Host: addr.String(),
	}

	dir := "testdata"
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory %q: %v", dir, err)
	}
	for _, file := range files {
		p := filepath.Join(dir, file.Name())
		if !file.IsDir() {
			t.Errorf("found non-directory %q", p)
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			p := filepath.Join(dir, file.Name())

			ctx := context.Background()

			objects, err := loadObjectsFromFilesystem(filepath.Join(p, "apply.yaml"))
			if err != nil {
				t.Fatalf("error loading objects: %v", err)
			}

			target, err := NewClusterTarget(restConfig)
			if err != nil {
				t.Fatalf("error building target: %v", err)
			}

			planner := &Planner{}

			plan, err := planner.BuildPlan(ctx, objects, target)
			if err != nil {
				t.Fatalf("error from BuildPlan: %v", err)
			}

			actual, err := yaml.Marshal(plan)
			if err != nil {
				t.Fatalf("yaml.Marshal failed: %v", err)
			}
			CompareGoldenFile(t, filepath.Join(p, "plan.yaml"), actual)
		})
	}
}

func CompareGoldenFile(t *testing.T, p string, got []byte) {
	if os.Getenv("WRITE_GOLDEN_OUTPUT") != "" {
		// Short-circuit when the output is correct
		b, err := os.ReadFile(p)
		if err == nil && bytes.Equal(b, got) {
			return
		}

		if err := os.WriteFile(p, got, 0644); err != nil {
			t.Fatalf("failed to write golden output %s: %v", p, err)
		}
		t.Errorf("wrote output to %s", p)
	} else {
		want, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("failed to read file %q: %v", p, err)
		}
		if diff := cmp.Diff(string(want), string(got)); diff != "" {
			t.Errorf("unexpected diff in %s: %s", p, diff)
		}
	}
}
