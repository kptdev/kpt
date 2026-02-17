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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestNewCELEvaluator(t *testing.T) {
	eval, err := NewCELEvaluator("true")
	require.NoError(t, err)
	assert.NotNil(t, eval)
	assert.NotNil(t, eval.env)
	assert.NotNil(t, eval.prg)
}

func TestNewCELEvaluator_EmptyCondition(t *testing.T) {
	eval, err := NewCELEvaluator("")
	require.NoError(t, err)
	assert.NotNil(t, eval)
	assert.NotNil(t, eval.env)
	assert.Nil(t, eval.prg)
}

func TestEvaluateCondition_EmptyCondition(t *testing.T) {
	eval, err := NewCELEvaluator("")
	require.NoError(t, err)

	result, err := eval.EvaluateCondition(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, result, "empty condition should return true")
}

func TestEvaluateCondition_SimpleTrue(t *testing.T) {
	eval, err := NewCELEvaluator("true")
	require.NoError(t, err)

	result, err := eval.EvaluateCondition(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluateCondition_SimpleFalse(t *testing.T) {
	eval, err := NewCELEvaluator("false")
	require.NoError(t, err)

	result, err := eval.EvaluateCondition(context.Background(), nil)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestEvaluateCondition_ResourceExists(t *testing.T) {
	// Create test resources
	configMapYAML := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  key: value
`
	deploymentYAML := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  replicas: 3
`

	configMap, err := yaml.Parse(configMapYAML)
	require.NoError(t, err)
	deployment, err := yaml.Parse(deploymentYAML)
	require.NoError(t, err)

	resources := []*yaml.RNode{configMap, deployment}

	// Test: ConfigMap exists
	condition := `resources.exists(r, r.kind == "ConfigMap" && r.metadata.name == "test-config")`
	eval, err := NewCELEvaluator(condition)
	require.NoError(t, err)
	result, err := eval.EvaluateCondition(context.Background(), resources)
	require.NoError(t, err)
	assert.True(t, result, "should find the ConfigMap")

	// Test: ConfigMap with wrong name doesn't exist
	condition = `resources.exists(r, r.kind == "ConfigMap" && r.metadata.name == "wrong-name")`
	eval, err = NewCELEvaluator(condition)
	require.NoError(t, err)
	result, err = eval.EvaluateCondition(context.Background(), resources)
	require.NoError(t, err)
	assert.False(t, result, "should not find ConfigMap withwrong name")

	// Test: Deployment exists
	condition = `resources.exists(r, r.kind == "Deployment")`
	eval, err = NewCELEvaluator(condition)
	require.NoError(t, err)
	result, err = eval.EvaluateCondition(context.Background(), resources)
	require.NoError(t, err)
	assert.True(t, result, "should find the Deployment")
}

func TestEvaluateCondition_ResourceCount(t *testing.T) {
	// Create test resources
	deploymentYAML := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  replicas: 3
`

	deployment, err := yaml.Parse(deploymentYAML)
	require.NoError(t, err)

	resources := []*yaml.RNode{deployment}

	// Test: Count of deployments is greater than 0
	condition := `resources.filter(r, r.kind == "Deployment").size() > 0`
	eval, err := NewCELEvaluator(condition)
	require.NoError(t, err)
	result, err := eval.EvaluateCondition(context.Background(), resources)
	require.NoError(t, err)
	assert.True(t, result, "should find deployments")

	// Test: Count of ConfigMaps is 0
	condition = `resources.filter(r, r.kind == "ConfigMap").size() == 0`
	eval, err = NewCELEvaluator(condition)
	require.NoError(t, err)
	result, err = eval.EvaluateCondition(context.Background(), resources)
	require.NoError(t, err)
	assert.True(t, result, "should not find ConfigMaps")
}

func TestEvaluateCondition_InvalidExpression(t *testing.T) {
	// Test invalid syntax
	_, err := NewCELEvaluator("this is not valid CEL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile")
}

func TestEvaluateCondition_NonBooleanResult(t *testing.T) {
	// Expression that returns a number, not a boolean
	_, err := NewCELEvaluator("1 + 1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must return a boolean")
}

// TestEvaluateCondition_Immutability ensures CEL evaluation cannot mutate the input resources
func TestEvaluateCondition_Immutability(t *testing.T) {
	configMapYAML := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  key: original-value
`

	configMap, err := yaml.Parse(configMapYAML)
	require.NoError(t, err)

	resources := []*yaml.RNode{configMap}

	// Store original values
	originalYAML, err := configMap.String()
	require.NoError(t, err)

	// Evaluate a condition that accesses the resources
	condition := `resources.exists(r, r.kind == "ConfigMap")`
	eval, err := NewCELEvaluator(condition)
	require.NoError(t, err)
	
	_, err = eval.EvaluateCondition(context.Background(), resources)
	require.NoError(t, err)

	// Verify resources haven't been mutated
	afterYAML, err := configMap.String()
	require.NoError(t, err)
	assert.Equal(t, originalYAML, afterYAML, "CEL evaluation should not mutate input resources")
}
