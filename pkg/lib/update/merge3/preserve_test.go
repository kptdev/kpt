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

package merge3

import (
	"testing"

	"github.com/kptdev/krm-functions-sdk/go/fn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/openapi"
)

const (
	originDeployment = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  template:
    spec:
      containers:
      - name: web
        image: nginx
`
	updatedDeployment = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  template:
    spec:
      containers:
      - name: web
        image: nginx:updated
`
	destNullContainers = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  template:
    spec:
      containers: null
`
	deploymentNoContainers = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  template:
    spec: {}
`
)

func mergeDeployments(t *testing.T, originYAML, updatedYAML, destYAML string, preserve bool) *fn.KubeObject {
	t.Helper()
	var origin fn.KubeObjects
	var err error
	if originYAML != "" {
		origin, err = fn.ParseKubeObjects([]byte(originYAML))
		require.NoError(t, err)
	}
	updated, err := fn.ParseKubeObjects([]byte(updatedYAML))
	require.NoError(t, err)
	dest, err := fn.ParseKubeObjects([]byte(destYAML))
	require.NoError(t, err)

	openapi.ResetOpenAPI()
	result, err := Merge(origin, updated, dest, nil, preserve)
	require.NoError(t, err)
	require.Len(t, result, 1)
	return result[0]
}

func TestPreserveExplicitNullAssociativeList(t *testing.T) {
	tests := []struct {
		name        string
		origin      string
		updated     string
		wantNull    bool
		wantContain string
	}{
		{
			name:     "dest-null list is kept when all sides present",
			origin:   originDeployment,
			updated:  updatedDeployment,
			wantNull: true,
		},
		{
			name:     "dest-null list is kept when origin omits the field",
			origin:   deploymentNoContainers,
			updated:  updatedDeployment,
			wantNull: true,
		},
		{
			name:     "dest-null list is kept when origin resource is missing",
			updated:  updatedDeployment,
			wantNull: true,
		},
		{
			name:     "dest-null list is kept when updated omits the field",
			origin:   originDeployment,
			updated:  deploymentNoContainers,
			wantNull: true,
		},
		{
			name:        "updated list wins when origin was already null",
			origin:      destNullContainers,
			updated:     updatedDeployment,
			wantContain: "nginx:updated",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeDeployments(t, tc.origin, tc.updated, destNullContainers, true)
			out := got.String()
			spec := got.GetMap("spec").GetMap("template").GetMap("spec")
			require.NotNil(t, spec)
			if tc.wantNull {
				assert.Regexp(t, `(?m)^\s*containers:\s*(null|~)?\s*$`, out, "expected containers to remain explicitly null, got:\n%s", out)
				assert.Nil(t, spec.GetSlice("containers"), "expected containers to stay null rather than become a list, got:\n%s", out)
				assert.NotContains(t, out, "image:")
				return
			}
			assert.Contains(t, out, tc.wantContain, "got:\n%s", out)
			assert.NotNil(t, spec.GetSlice("containers"))
		})
	}
}

func TestAssociativeListDestNullDeletedByDefault(t *testing.T) {
	origin, err := fn.ParseKubeObject([]byte(originDeployment))
	require.NoError(t, err)
	updated, err := fn.ParseKubeObject([]byte(updatedDeployment))
	require.NoError(t, err)
	dest, err := fn.ParseKubeObject([]byte(destNullContainers))
	require.NoError(t, err)

	openapi.ResetOpenAPI()
	result, err := Merge(fn.KubeObjects{origin}, fn.KubeObjects{updated}, fn.KubeObjects{dest}, nil, false)
	require.NoError(t, err)
	require.Len(t, result, 1)

	out := result[0].String()
	spec := result[0].GetMap("spec").GetMap("template").GetMap("spec")
	require.NotNil(t, spec)
	assert.Nil(t, spec.GetSlice("containers"), "expected dest-null containers to be removed by default, got:\n%s", out)
	assert.NotContains(t, out, "nginx")
}
