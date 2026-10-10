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

package kptops

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	fnresultv1 "github.com/kptdev/kpt/api/fnresult/v1"
	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/kptdev/kpt/pkg/fn"
	"github.com/kptdev/kpt/pkg/lib/runneroptions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func TestStatusRenderer_Render_ReturnsRenderStatus(t *testing.T) {
	kptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-app
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
      configMap:
        namespace: production
`
	resources := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 1
`

	pkgPath := t.TempDir()

	fs := filesys.MakeFsOnDisk()
	require.NoError(t, fs.WriteFile(filepath.Join(pkgPath, "Kptfile"), []byte(kptfile)))
	require.NoError(t, fs.WriteFile(filepath.Join(pkgPath, "resources.yaml"), []byte(resources)))

	opts := runneroptions.RunnerOptions{
		ImagePullPolicy: runneroptions.IfNotPresentPull,
	}
	opts.InitDefaults(runneroptions.GHCRImagePrefix)
	require.NoError(t, opts.InitCELEnvironment())

	renderer := NewStatusRenderer(opts)
	require.NotNil(t, renderer)

	status, err := renderer.Render(context.Background(), fs, fn.RenderOptions{
		PkgPath: pkgPath,
		Runtime: &testRuntime{},
	})

	require.NoError(t, err)
	require.NotNil(t, status)
	assert.IsType(t, &kptfilev1.RenderStatus{}, status)
	assert.Len(t, status.MutationSteps, 1)
	assert.Equal(t, "ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest", status.MutationSteps[0].Image)
	assert.Empty(t, status.ErrorSummary)
}

func TestStatusRenderer_Render_ReturnsRenderStatusOnError(t *testing.T) {
	kptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-app
pipeline:
  mutators:
    - image: error-func:latest
`
	resources := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
`

	pkgPath := t.TempDir()

	fs := filesys.MakeFsOnDisk()
	require.NoError(t, fs.WriteFile(filepath.Join(pkgPath, "Kptfile"), []byte(kptfile)))
	require.NoError(t, fs.WriteFile(filepath.Join(pkgPath, "resources.yaml"), []byte(resources)))

	opts := runneroptions.RunnerOptions{
		ImagePullPolicy: runneroptions.IfNotPresentPull,
	}
	opts.InitDefaults(runneroptions.GHCRImagePrefix)
	require.NoError(t, opts.InitCELEnvironment())

	renderer := NewStatusRenderer(opts)
	require.NotNil(t, renderer)

	status, err := renderer.Render(context.Background(), fs, fn.RenderOptions{
		PkgPath: pkgPath,
		Runtime: &mockFnRuntime{shouldFail: true},
	})

	require.Error(t, err)
	require.NotNil(t, status)
	assert.NotEmpty(t, status.ErrorSummary)
}

func TestStatusRenderer_Render_PreHydrationError(t *testing.T) {
	pkgPath := t.TempDir()
	fs := filesys.MakeFsOnDisk()
	require.NoError(t, fs.WriteFile(filepath.Join(pkgPath, "Kptfile"), []byte("invalid: [yaml content: {")))

	opts := runneroptions.RunnerOptions{
		ImagePullPolicy: runneroptions.IfNotPresentPull,
	}
	opts.InitDefaults(runneroptions.GHCRImagePrefix)
	require.NoError(t, opts.InitCELEnvironment())

	renderer := NewStatusRenderer(opts)
	require.NotNil(t, renderer)

	status, err := renderer.Render(context.Background(), fs, fn.RenderOptions{
		PkgPath: pkgPath,
		Runtime: &testRuntime{},
	})

	require.Error(t, err)
	require.NotNil(t, status)
	assert.NotEmpty(t, status.ErrorSummary)
	assert.Contains(t, status.ErrorSummary, "Kptfile")
}

func TestLegacyRenderer_Render(t *testing.T) {
	kptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-app
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
      configMap:
        namespace: production
`
	resources := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 1
`

	pkgPath := t.TempDir()

	fs := filesys.MakeFsOnDisk()
	require.NoError(t, fs.WriteFile(filepath.Join(pkgPath, "Kptfile"), []byte(kptfile)))
	require.NoError(t, fs.WriteFile(filepath.Join(pkgPath, "resources.yaml"), []byte(resources)))

	opts := runneroptions.RunnerOptions{
		ImagePullPolicy: runneroptions.IfNotPresentPull,
	}
	opts.InitDefaults(runneroptions.GHCRImagePrefix)
	require.NoError(t, opts.InitCELEnvironment())

	//nolint:staticcheck // SA1019: NewRenderer is deprecated, testing backward compatibility
	legacy := NewRenderer(opts)
	require.NotNil(t, legacy)

	results, err := legacy.Render(context.Background(), fs, fn.RenderOptions{
		PkgPath: pkgPath,
		Runtime: &testRuntime{},
	})

	require.NoError(t, err)
	require.NotNil(t, results)
	assert.IsType(t, &fnresultv1.ResultList{}, results)
}

func TestBuildRenderStatus_NoStepsWithError(t *testing.T) {
	hctx := &hydrationContext{}
	testErr := fmt.Errorf("pre-hydration error: failed to read Kptfile")
	status := buildRenderStatus(hctx, testErr)
	require.NotNil(t, status)
	assert.Equal(t, testErr.Error(), status.ErrorSummary)
	assert.Empty(t, status.MutationSteps)
	assert.Empty(t, status.ValidationSteps)
}

func TestBuildRenderStatus_NoStepsNilError(t *testing.T) {
	hctx := &hydrationContext{}
	status := buildRenderStatus(hctx, nil)
	assert.Nil(t, status)
}
