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

package runneroptions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestEvaluateCondition_ResourceToMap(t *testing.T) {
	env, err := NewCELEnvironment()
	require.NoError(t, err)

	resource, err := yaml.Parse("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\ndata:\n  key: val\n")
	require.NoError(t, err)

	// Exercises the refactored resourceToMap (RNode.Map() + ensureMetadata) path
	result, err := env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.kind == "ConfigMap" && r.metadata.name == "test")`,
		[]*yaml.RNode{resource}, 100, 1000000)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluateCondition_EnsureMetadataDefaults(t *testing.T) {
	env, err := NewCELEnvironment()
	require.NoError(t, err)

	// Resource missing metadata — exercises ensureMetadata defaulting
	resource, err := yaml.Parse("apiVersion: v1\nkind: ConfigMap\ndata:\n  key: val\n")
	require.NoError(t, err)

	result, err := env.EvaluateCondition(context.Background(),
		`resources.exists(r, r.metadata.name == "")`,
		[]*yaml.RNode{resource}, 100, 1000000)
	require.NoError(t, err)
	assert.True(t, result)
}
