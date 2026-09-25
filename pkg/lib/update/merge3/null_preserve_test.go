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
	"fmt"
	"testing"

	"github.com/kptdev/krm-functions-sdk/go/fn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/openapi"
)

const (
	configMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key1: %s
  key2: %s
`

	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  template:
    spec: %s`
)

var (
	originDeployment       = fmt.Sprintf(deploymentTemplate, "\n      containers:\n      - name: web\n        image: nginx")
	updatedDeployment      = fmt.Sprintf(deploymentTemplate, "\n      containers:\n      - name: web\n        image: nginx:updated")
	destNullContainers     = fmt.Sprintf(deploymentTemplate, "\n      containers: null")
	deploymentNoContainers = fmt.Sprintf(deploymentTemplate, "{}")
)

// key1 is the one we are mainly testing, key2 is just for control
type twoKeyPair struct {
	key1, key2 string
}

func (tk *twoKeyPair) Templated() string {
	return fmt.Sprintf(configMapTemplate, tk.key1, tk.key2)
}

func TestPreserveExplicitNull(t *testing.T) {
	testCases := map[string]struct {
		orig, upstream, dest, expected twoKeyPair
	}{
		"original null unchanged": {
			orig:     twoKeyPair{key1: "null", key2: "value2"},
			upstream: twoKeyPair{key1: "null", key2: "newvalue2"},
			dest:     twoKeyPair{key1: "null", key2: "value2"},

			expected: twoKeyPair{key1: "null", key2: "newvalue2"},
		},
		"upstream null preserved": {
			orig:     twoKeyPair{key1: "value1", key2: "value2"},
			upstream: twoKeyPair{key1: "null", key2: "value2"},
			dest:     twoKeyPair{key1: "newvalue1", key2: "newvalue2"},

			expected: twoKeyPair{key1: "null", key2: "newvalue2"},
		},
		"destination null preserved": {
			orig:     twoKeyPair{key1: "value1", key2: "value2"},
			upstream: twoKeyPair{key1: "value1", key2: "newvalue2"},
			dest:     twoKeyPair{key1: "null", key2: "value2"},

			expected: twoKeyPair{key1: "null", key2: "newvalue2"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mergedKo := mergeYamls(t,
				tc.orig.Templated(),
				tc.upstream.Templated(),
				tc.dest.Templated(),
				true,
			)

			assert.Equal(t, tc.expected.Templated(), mergedKo.String())
		})
	}
}

func TestPreserveImplicitNull(t *testing.T) {
	testCases := map[string]struct {
		orig, upstream, dest, expected twoKeyPair
	}{
		"original null unchanged": {
			orig:     twoKeyPair{key1: "", key2: "value2"},
			upstream: twoKeyPair{key1: "", key2: "newvalue2"},
			dest:     twoKeyPair{key1: "", key2: "value2"},

			expected: twoKeyPair{key1: "", key2: "newvalue2"},
		},
		"upstream null preserved": {
			orig:     twoKeyPair{key1: "value1", key2: "value2"},
			upstream: twoKeyPair{key1: "", key2: "value2"},
			dest:     twoKeyPair{key1: "newvalue1", key2: "newvalue2"},

			expected: twoKeyPair{key1: "", key2: "newvalue2"},
		},
		"destination null preserved": {
			orig:     twoKeyPair{key1: "value1", key2: "value2"},
			upstream: twoKeyPair{key1: "value1", key2: "newvalue2"},
			dest:     twoKeyPair{key1: "", key2: "value2"},

			expected: twoKeyPair{key1: "", key2: "newvalue2"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mergedKo := mergeYamls(t,
				tc.orig.Templated(),
				tc.upstream.Templated(),
				tc.dest.Templated(),
				true,
			)

			assert.Equal(t, tc.expected.Templated(), mergedKo.String())
		})
	}
}

func mergeYamls(t *testing.T, originYAML, updatedYAML, destYAML string, preserve bool) *fn.KubeObject {
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
			got := mergeYamls(t, tc.origin, tc.updated, destNullContainers, true)
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
