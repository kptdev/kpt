// Copyright 2026 The kpt and Nephio Authors
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

package fnruntime

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

func TestNewCELEnvironment(t *testing.T) {
	env := newTestEnv(t)
	assert.NotNil(t, env)
}

func TestEvaluateCondition_EmptyCondition(t *testing.T) {
	env := newTestEnv(t)
	result, err := env.EvaluateCondition(context.Background(), "", nil)
	require.NoError(t, err)
	assert.True(t, result, "empty condition should return true")
}

func TestEvaluateCondition_SimpleTrue(t *testing.T) {
	env := newTestEnv(t)
	result, err := env.EvaluateCondition(context.Background(), "true", nil)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluateCondition_SimpleFalse(t *testing.T) {
	env := newTestEnv(t)
	result, err := env.EvaluateCondition(context.Background(), "false", nil)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestEvaluateCondition_ResourceExists(t *testing.T) {
	env := newTestEnv(t)

	configMap, err := yaml.Parse("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test-config\ndata:\n  key: value")
	require.NoError(t, err)
	deployment, err := yaml.Parse("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: test-deployment\nspec:\n  replicas: 3")
	require.NoError(t, err)

	resources := []*yaml.RNode{configMap, deployment}

	result, err := env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.kind == "ConfigMap" && r.metadata.name == "test-config")`, resources)
	require.NoError(t, err)
	assert.True(t, result)

	result, err = env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.kind == "ConfigMap" && r.metadata.name == "wrong-name")`, resources)
	require.NoError(t, err)
	assert.False(t, result)

	result, err = env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.kind == "Deployment")`, resources)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluateCondition_ResourceCount(t *testing.T) {
	env := newTestEnv(t)

	deployment, err := yaml.Parse("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: test-deployment\nspec:\n  replicas: 3")
	require.NoError(t, err)
	resources := []*yaml.RNode{deployment}

	result, err := env.EvaluateCondition(context.Background(),
		`resources.filter(r, r.kind == "Deployment").size() > 0`, resources)
	require.NoError(t, err)
	assert.True(t, result)

	result, err = env.EvaluateCondition(context.Background(),
		`resources.filter(r, r.kind == "ConfigMap").size() == 0`, resources)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluateCondition_InvalidExpression(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.EvaluateCondition(context.Background(), "this is not valid CEL", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile")
}

func TestEvaluateCondition_NonBooleanResult(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.EvaluateCondition(context.Background(), "1 + 1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must return a boolean")
}

func TestEvaluateCondition_Immutability(t *testing.T) {
	env := newTestEnv(t)

	configMap, err := yaml.Parse("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test-config\n  namespace: default\ndata:\n  key: original-value")
	require.NoError(t, err)

	originalYAML, err := configMap.String()
	require.NoError(t, err)

	_, err = env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.kind == "ConfigMap")`, []*yaml.RNode{configMap})
	require.NoError(t, err)

	afterYAML, err := configMap.String()
	require.NoError(t, err)
	assert.Equal(t, originalYAML, afterYAML, "CEL evaluation should not mutate input resources")
}
