// Copyright 2021 The kpt Authors
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
	"testing"

	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestIsMatch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		selector kptfile.Selector
		expected bool
	}{
		{
			name: "kind match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				Kind: "Deployment",
			},
			expected: true,
		},
		{
			name: "name match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				Name: "nginx-deployment",
			},
			expected: true,
		},
		{
			name: "namespace match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				Namespace: "staging",
			},
			expected: true,
		},
		{
			name: "apiVersion match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				APIVersion: "apps/v1",
			},
			expected: true,
		},
		{
			name: "GVKNN match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				Name:       "nginx-deployment",
				Namespace:  "staging",
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			expected: true,
		},
		{
			name: "namespace not matched but rest did",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfile.Selector{
				Name:       "nginx-deployment",
				Namespace:  "prod",
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			expected: false,
		},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			node, err := yaml.Parse(tc.input)
			assert.NoError(t, err)
			actual := IsMatch(node, tc.selector)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestNewConfigMap(t *testing.T) {
	data := map[string]string{
		"normal string": "abc",
		"integer":       "8081",
		"float":         "1.23",
		"bool":          "true",
	}
	m, err := NewConfigMap(data)
	assert.NoError(t, err)
	mapAsString := m.MustString()
	assert.Contains(t, mapAsString, `bool: "true"`)
	assert.Contains(t, mapAsString, `normal string: abc`)
	assert.Contains(t, mapAsString, `integer: "8081"`)
	assert.Contains(t, mapAsString, `float: "1.23"`)
}
