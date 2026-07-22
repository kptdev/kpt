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

package runtime

import (
	"context"
	"testing"

	"github.com/kptdev/kpt/pkg/lib/runneroptions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func newTestEnv(t *testing.T) *runneroptions.CELEnvironment {
	t.Helper()
	env, err := runneroptions.NewCELEnvironment()
	require.NoError(t, err)
	return env
}

func parseResource(t *testing.T, content string) *yaml.RNode {
	t.Helper()
	node, err := yaml.Parse(content)
	require.NoError(t, err)
	return node
}

func TestNewCELEnvironment(t *testing.T) {
	env := newTestEnv(t)
	assert.NotNil(t, env)
}

func TestEvaluateCondition_BasicExpressions(t *testing.T) {
	env := newTestEnv(t)
	testCases := []struct {
		name      string
		expr      string
		expectVal bool
		expectErr bool
		errMsg    string
	}{
		{name: "empty condition", expr: "", expectVal: true},
		{name: "simple true", expr: "true", expectVal: true},
		{name: "simple false", expr: "false", expectVal: false},
		{name: "invalid expression", expr: "this is not valid CEL", expectErr: true, errMsg: "failed to compile"},
		{name: "non-boolean result", expr: "1 + 1", expectErr: true, errMsg: "must return a boolean"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := env.EvaluateCondition(context.Background(), tc.expr, nil, 100, 1000000)
			if tc.expectErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectVal, res)
			}
		})
	}
}

func TestEvaluateCondition_ResourceExists(t *testing.T) {
	env := newTestEnv(t)

	configMap := parseResource(t, "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm-item\ndata:\n  setting: enabled")
	deployment := parseResource(t, "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: web-app\nspec:\n  replicas: 5")

	resources := []*yaml.RNode{configMap, deployment}

	result, err := env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.kind == "ConfigMap" && r.metadata.name == "cm-item")`, resources, 100, 1000000)
	require.NoError(t, err)
	assert.True(t, result)

	result, err = env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.kind == "ConfigMap" && r.metadata.name == "other-cm")`, resources, 100, 1000000)
	require.NoError(t, err)
	assert.False(t, result)

	result, err = env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.kind == "Deployment")`, resources, 100, 1000000)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluateCondition_ResourceCount(t *testing.T) {
	env := newTestEnv(t)

	deployment := parseResource(t, "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api-deploy\nspec:\n  replicas: 2")
	resources := []*yaml.RNode{deployment}

	result, err := env.EvaluateCondition(context.Background(),
		`resources.filter(r, r.kind == "Deployment").size() > 0`, resources, 100, 1000000)
	require.NoError(t, err)
	assert.True(t, result)

	result, err = env.EvaluateCondition(context.Background(),
		`resources.filter(r, r.kind == "ConfigMap").size() == 0`, resources, 100, 1000000)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluateCondition_Immutability(t *testing.T) {
	env := newTestEnv(t)

	cm := parseResource(t, "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: immutable-cm\n  namespace: sys\ndata:\n  foo: bar")
	originalYAML, err := cm.String()
	require.NoError(t, err)

	_, err = env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.kind == "ConfigMap")`, []*yaml.RNode{cm}, 100, 1000000)
	require.NoError(t, err)

	afterYAML, err := cm.String()
	require.NoError(t, err)
	assert.Equal(t, originalYAML, afterYAML, "CEL evaluation must preserve input resource immutability")
}

func TestEvaluateCondition_MissingMetadata(t *testing.T) {
	env := newTestEnv(t)

	noMetadata := parseResource(t, "apiVersion: v1\nkind: ConfigMap\ndata:\n  key: val")
	noName := parseResource(t, "apiVersion: v1\nkind: ConfigMap\nmetadata: {}\ndata:\n  key: val2")
	resources := []*yaml.RNode{noMetadata, noName}

	result, err := env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.kind == "ConfigMap" && r.metadata.name == "cm-item")`, resources, 100, 1000000)
	require.NoError(t, err)
	assert.False(t, result)

	result, err = env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.kind == "ConfigMap")`, resources, 100, 1000000)
	require.NoError(t, err)
	assert.True(t, result)
}
