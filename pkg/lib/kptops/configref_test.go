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
	"testing"

	"github.com/kptdev/kpt/pkg/lib/runneroptions"
	"github.com/kptdev/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

const testDeploymentResource = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 3
`

// configRefTestSetup holds the filesystem and renderer for configRef tests.
type configRefTestSetup struct {
	fs filesys.FileSystem
	r  Renderer
}

// setupConfigRefTest creates an in-memory filesystem populated with the given
// Kptfile, optional function config, and the standard test deployment resource,
// then returns a ready-to-use Renderer. Pass an empty string for fnConfig to
// skip writing fn-config.yaml.
func setupConfigRefTest(t *testing.T, kptfile, fnConfig string) configRefTestSetup {
	t.Helper()

	fs := filesys.MakeFsInMemory()
	require.NoError(t, fs.MkdirAll("/pkg"))
	require.NoError(t, fs.WriteFile("/pkg/Kptfile", []byte(kptfile)))
	if fnConfig != "" {
		require.NoError(t, fs.WriteFile("/pkg/fn-config.yaml", []byte(fnConfig)))
	}
	require.NoError(t, fs.WriteFile("/pkg/resources.yaml", []byte(testDeploymentResource)))

	r := Renderer{
		PkgPath:    "/pkg",
		FileSystem: fs,
	}
	r.RunnerOptions.InitDefaults(runneroptions.GHCRImagePrefix)
	r.RunnerOptions.ImagePullPolicy = runneroptions.IfNotPresentPull

	return configRefTestSetup{fs: fs, r: r}
}

func TestRenderWithConfigRef_Mutator(t *testing.T) {
	kptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-pkg
  annotations:
    config.kubernetes.io/local-config: "true"
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
      configRef:
        kind: ConfigMap
        name: ns-config
`

	fnConfig := `apiVersion: v1
kind: ConfigMap
metadata:
  name: ns-config
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  namespace: production
`

	ts := setupConfigRefTest(t, kptfile, fnConfig)

	_, err := ts.r.Execute(fake.CtxWithDefaultPrinter())
	require.NoError(t, err)

	// Verify the deployment got the namespace set
	res, err := ts.fs.ReadFile("/pkg/resources.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(res), "namespace: production")
}

func TestRenderWithConfigRef_Validator(t *testing.T) {
	// Validators run on a copy but shouldn't error if configRef resolves
	kptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-pkg
  annotations:
    config.kubernetes.io/local-config: "true"
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
      configMap:
        namespace: staging
  validators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
      configRef:
        kind: ConfigMap
        name: label-config
`

	fnConfig := `apiVersion: v1
kind: ConfigMap
metadata:
  name: label-config
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  env: staging
`

	ts := setupConfigRefTest(t, kptfile, fnConfig)

	_, err := ts.r.Execute(fake.CtxWithDefaultPrinter())
	require.NoError(t, err)

	// Verify the mutator ran (namespace set) and the validator didn't error
	res, err := ts.fs.ReadFile("/pkg/resources.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(res), "namespace: staging")
}

func TestRenderWithConfigRef_NotFound(t *testing.T) {
	// configRef pointing to a nonexistent resource should fail
	kptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-pkg
  annotations:
    config.kubernetes.io/local-config: "true"
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
      configRef:
        kind: ConfigMap
        name: does-not-exist
`

	ts := setupConfigRefTest(t, kptfile, "")

	_, err := ts.r.Execute(fake.CtxWithDefaultPrinter())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does-not-exist")
}

func TestRenderWithConfigRef_APIVersionFilter(t *testing.T) {
	// configRef with apiVersion should match only resources with that apiVersion
	kptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-pkg
  annotations:
    config.kubernetes.io/local-config: "true"
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
      configRef:
        apiVersion: v1
        kind: ConfigMap
        name: ns-config
`

	fnConfig := `apiVersion: v1
kind: ConfigMap
metadata:
  name: ns-config
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  namespace: filtered
`

	ts := setupConfigRefTest(t, kptfile, fnConfig)

	_, err := ts.r.Execute(fake.CtxWithDefaultPrinter())
	require.NoError(t, err)

	res, err := ts.fs.ReadFile("/pkg/resources.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(res), "namespace: filtered")
}

func TestRenderWithConfigRef_MutatorChain(t *testing.T) {
	// Test that a configRef resource can be mutated by an earlier pipeline step
	// and the later step sees the updated config.
	kptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-pkg
  annotations:
    config.kubernetes.io/local-config: "true"
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:latest
      configMap:
        tier: backend
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
      configRef:
        kind: ConfigMap
        name: ns-config
`

	// This ConfigMap will be passed through the first mutator (set-labels),
	// which will add a label to it, then used as config for the second mutator.
	fnConfig := `apiVersion: v1
kind: ConfigMap
metadata:
  name: ns-config
data:
  namespace: chained
`

	ts := setupConfigRefTest(t, kptfile, fnConfig)

	_, err := ts.r.Execute(fake.CtxWithDefaultPrinter())
	require.NoError(t, err)

	// Verify the second mutator got the config and set the namespace
	res, err := ts.fs.ReadFile("/pkg/resources.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(res), "namespace: chained")
}

func TestRenderWithConfigRef_AmbiguousMatch(t *testing.T) {
	// configRef that matches multiple resources should fail at validation
	kptfile := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-pkg
  annotations:
    config.kubernetes.io/local-config: "true"
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
      configRef:
        kind: ConfigMap
        name: ns-config
`

	// Two ConfigMaps with the same name but different namespaces
	fnConfig := `apiVersion: v1
kind: ConfigMap
metadata:
  name: ns-config
  namespace: ns-a
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  namespace: production
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ns-config
  namespace: ns-b
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  namespace: staging
`

	ts := setupConfigRefTest(t, kptfile, fnConfig)

	_, err := ts.r.Execute(fake.CtxWithDefaultPrinter())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "matched 2 resources")
}
