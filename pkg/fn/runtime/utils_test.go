// Copyright 2021,2026 The kpt Authors
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
	"testing"

	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// helper to build a node with annotations for package/path
func makeNode(t *testing.T, apiVersion, kind, name, namespace string, labels, annotations map[string]string, pkgPath, filePath string) *yaml.RNode {
	t.Helper()
	node := yaml.MustParse(`apiVersion: v1
kind: ConfigMap
metadata:
  name: placeholder
`)
	if err := node.PipeE(yaml.SetField("apiVersion", yaml.NewStringRNode(apiVersion))); err != nil {
		t.Fatal(err)
	}
	if err := node.PipeE(yaml.SetField("kind", yaml.NewStringRNode(kind))); err != nil {
		t.Fatal(err)
	}
	if err := node.SetName(name); err != nil {
		t.Fatal(err)
	}
	if namespace != "" {
		if err := node.SetNamespace(namespace); err != nil {
			t.Fatal(err)
		}
	}
	if len(labels) > 0 {
		if err := node.SetLabels(labels); err != nil {
			t.Fatal(err)
		}
	}
	for k, v := range annotations {
		if err := node.PipeE(yaml.SetAnnotation(k, v)); err != nil {
			t.Fatal(err)
		}
	}
	if pkgPath != "" {
		if err := node.PipeE(yaml.SetAnnotation(PackagePathAnnotation, pkgPath)); err != nil {
			t.Fatal(err)
		}
	}
	if filePath != "" {
		if err := node.PipeE(yaml.SetAnnotation(PathAnnotation, filePath)); err != nil {
			t.Fatal(err)
		}
	}
	return node
}

func TestSelectInput(t *testing.T) {
	ctx := &SelectionContext{RootPackagePath: "/root"}

	deployment := makeNode(t, "apps/v1", "Deployment", "my-deploy", "default",
		map[string]string{"env": "prod"}, nil, "/root", "deploy.yaml")
	service := makeNode(t, "v1", "Service", "my-svc", "default",
		map[string]string{"env": "prod"}, nil, "/root", "service.yaml")
	configmap := makeNode(t, "v1", "ConfigMap", "my-cm", "staging",
		map[string]string{"env": "staging"}, nil, "/root", "cm.yaml")

	input := []*yaml.RNode{deployment, service, configmap}

	tests := []struct {
		name       string
		selectors  []kptfilev1.Selector
		exclusions []kptfilev1.Selector
		wantNames  []string
		wantErr    bool
	}{
		{
			name:      "no selectors no exclusions returns all",
			wantNames: []string{"my-deploy", "my-svc", "my-cm"},
		},
		{
			name:      "selector by kind",
			selectors: []kptfilev1.Selector{{Kind: "Deployment"}},
			wantNames: []string{"my-deploy"},
		},
		{
			name:      "selector by label",
			selectors: []kptfilev1.Selector{{Labels: map[string]string{"env": "prod"}}},
			wantNames: []string{"my-deploy", "my-svc"},
		},
		{
			name:       "selector all, exclude by kind",
			exclusions: []kptfilev1.Selector{{Kind: "Service"}},
			wantNames:  []string{"my-deploy", "my-cm"},
		},
		{
			name:       "selector by label, exclude by namespace",
			selectors:  []kptfilev1.Selector{{Labels: map[string]string{"env": "prod"}}},
			exclusions: []kptfilev1.Selector{{Namespace: "default"}},
			wantNames:  []string{},
		},
		{
			name:      "selector by regexp",
			selectors: []kptfilev1.Selector{{ResourceFileRegexp: `deploy\.yaml`}},
			wantNames: []string{"my-deploy"},
		},
		{
			name:      "invalid regexp returns error",
			selectors: []kptfilev1.Selector{{ResourceFileRegexp: `[invalid`}},
			wantErr:   true,
		},
		{
			name:       "invalid regexp in exclusion returns error",
			exclusions: []kptfilev1.Selector{{ResourceFileRegexp: `[invalid`}},
			wantErr:    true,
		},
		{
			name:       "empty exclusion selector is skipped",
			exclusions: []kptfilev1.Selector{{}},
			wantNames:  []string{"my-deploy", "my-svc", "my-cm"},
		},
		{
			name:      "multiple selectors union",
			selectors: []kptfilev1.Selector{{Kind: "Deployment"}, {Kind: "Service"}},
			wantNames: []string{"my-deploy", "my-svc"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SelectInput(input, tc.selectors, tc.exclusions, ctx)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			var gotNames []string
			for _, n := range got {
				gotNames = append(gotNames, n.GetName())
			}
			if tc.wantNames == nil {
				tc.wantNames = []string{}
			}
			if gotNames == nil {
				gotNames = []string{}
			}
			assert.ElementsMatch(t, tc.wantNames, gotNames)
		})
	}
}

func TestGetSelectedInput(t *testing.T) {
	ctx := &SelectionContext{RootPackagePath: "/root"}

	deployment := makeNode(t, "apps/v1", "Deployment", "deploy", "default", nil, nil, "/root", "deploy.yaml")
	service := makeNode(t, "v1", "Service", "svc", "default", nil, nil, "/root", "svc.yaml")
	input := []*yaml.RNode{deployment, service}

	tests := []struct {
		name      string
		selectors []kptfilev1.Selector
		wantNames []string
		wantErr   bool
	}{
		{
			name:      "empty selectors returns all",
			wantNames: []string{"deploy", "svc"},
		},
		{
			name:      "select by kind",
			selectors: []kptfilev1.Selector{{Kind: "Service"}},
			wantNames: []string{"svc"},
		},
		{
			name:      "no match returns empty",
			selectors: []kptfilev1.Selector{{Kind: "DaemonSet"}},
			wantNames: []string{},
		},
		{
			name:      "invalid regexp returns error",
			selectors: []kptfilev1.Selector{{ResourceFileRegexp: `[bad`}},
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getSelectedInput(input, tc.selectors, ctx)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			var gotNames []string
			for _, n := range got {
				gotNames = append(gotNames, n.GetName())
			}
			if gotNames == nil {
				gotNames = []string{}
			}
			assert.ElementsMatch(t, tc.wantNames, gotNames)
		})
	}
}

func TestGetInputNotExcluded(t *testing.T) {
	ctx := &SelectionContext{RootPackagePath: "/root"}

	deployment := makeNode(t, "apps/v1", "Deployment", "deploy", "default", nil, nil, "/root", "deploy.yaml")
	service := makeNode(t, "v1", "Service", "svc", "staging", nil, nil, "/root", "svc.yaml")
	input := []*yaml.RNode{deployment, service}

	tests := []struct {
		name       string
		exclusions []kptfilev1.Selector
		wantNames  []string
		wantErr    bool
	}{
		{
			name:      "no exclusions returns all",
			wantNames: []string{"deploy", "svc"},
		},
		{
			name:       "exclude by kind",
			exclusions: []kptfilev1.Selector{{Kind: "Deployment"}},
			wantNames:  []string{"svc"},
		},
		{
			name:       "exclude by namespace",
			exclusions: []kptfilev1.Selector{{Namespace: "staging"}},
			wantNames:  []string{"deploy"},
		},
		{
			name:       "exclude all",
			exclusions: []kptfilev1.Selector{{Kind: "Deployment"}, {Kind: "Service"}},
			wantNames:  []string{},
		},
		{
			name:       "empty exclusion selector is skipped",
			exclusions: []kptfilev1.Selector{{}},
			wantNames:  []string{"deploy", "svc"},
		},
		{
			name:       "invalid regexp returns error",
			exclusions: []kptfilev1.Selector{{ResourceFileRegexp: `[bad`}},
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getInputNotExcluded(input, tc.exclusions, ctx)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			var gotNames []string
			for _, n := range got {
				gotNames = append(gotNames, n.GetName())
			}
			if gotNames == nil {
				gotNames = []string{}
			}
			assert.ElementsMatch(t, tc.wantNames, gotNames)
		})
	}
}

func TestIsMatch(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		selector kptfilev1.Selector
		expected bool
	}{
		{
			name: "kind match",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.kubernetes.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfilev1.Selector{
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
    internal.config.kubernetes.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfilev1.Selector{
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
    internal.config.kubernetes.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfilev1.Selector{
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
    internal.config.kubernetes.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfilev1.Selector{
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
    internal.config.kubernetes.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfilev1.Selector{
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
    internal.config.kubernetes.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selector: kptfilev1.Selector{
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
			actual, _ := IsMatch(node, tc.selector, nil)
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

func TestIsMatchComprehensive(t *testing.T) {
	ctx := &SelectionContext{RootPackagePath: "/root"}

	tests := []struct {
		name     string
		node     *yaml.RNode
		selector kptfilev1.Selector
		want     bool
		wantErr  bool
	}{
		{
			name:     "empty selector matches everything",
			node:     makeNode(t, "apps/v1", "Deployment", "d", "ns", nil, nil, "", ""),
			selector: kptfilev1.Selector{},
			want:     true,
		},
		{
			name:     "kind match",
			node:     makeNode(t, "apps/v1", "Deployment", "d", "", nil, nil, "", ""),
			selector: kptfilev1.Selector{Kind: "Deployment"},
			want:     true,
		},
		{
			name:     "kind no match",
			node:     makeNode(t, "apps/v1", "Deployment", "d", "", nil, nil, "", ""),
			selector: kptfilev1.Selector{Kind: "Service"},
			want:     false,
		},
		{
			name:     "name match",
			node:     makeNode(t, "v1", "Service", "my-svc", "", nil, nil, "", ""),
			selector: kptfilev1.Selector{Name: "my-svc"},
			want:     true,
		},
		{
			name:     "name no match",
			node:     makeNode(t, "v1", "Service", "my-svc", "", nil, nil, "", ""),
			selector: kptfilev1.Selector{Name: "other-svc"},
			want:     false,
		},
		{
			name:     "namespace match",
			node:     makeNode(t, "v1", "Service", "svc", "prod", nil, nil, "", ""),
			selector: kptfilev1.Selector{Namespace: "prod"},
			want:     true,
		},
		{
			name:     "namespace no match",
			node:     makeNode(t, "v1", "Service", "svc", "prod", nil, nil, "", ""),
			selector: kptfilev1.Selector{Namespace: "staging"},
			want:     false,
		},
		{
			name:     "apiVersion match",
			node:     makeNode(t, "apps/v1", "Deployment", "d", "", nil, nil, "", ""),
			selector: kptfilev1.Selector{APIVersion: "apps/v1"},
			want:     true,
		},
		{
			name:     "apiVersion no match",
			node:     makeNode(t, "apps/v1", "Deployment", "d", "", nil, nil, "", ""),
			selector: kptfilev1.Selector{APIVersion: "apps/v2"},
			want:     false,
		},
		{
			name: "label match single",
			node: makeNode(t, "v1", "ConfigMap", "cm", "", map[string]string{"env": "prod"}, nil, "", ""),
			selector: kptfilev1.Selector{
				Labels: map[string]string{"env": "prod"},
			},
			want: true,
		},
		{
			name: "label match multiple",
			node: makeNode(t, "v1", "ConfigMap", "cm", "", map[string]string{"env": "prod", "team": "infra"}, nil, "", ""),
			selector: kptfilev1.Selector{
				Labels: map[string]string{"env": "prod", "team": "infra"},
			},
			want: true,
		},
		{
			name: "label no match wrong value",
			node: makeNode(t, "v1", "ConfigMap", "cm", "", map[string]string{"env": "staging"}, nil, "", ""),
			selector: kptfilev1.Selector{
				Labels: map[string]string{"env": "prod"},
			},
			want: false,
		},
		{
			name: "label no match missing key",
			node: makeNode(t, "v1", "ConfigMap", "cm", "", map[string]string{"team": "infra"}, nil, "", ""),
			selector: kptfilev1.Selector{
				Labels: map[string]string{"env": "prod"},
			},
			want: false,
		},
		{
			name: "annotation match",
			node: makeNode(t, "v1", "ConfigMap", "cm", "", nil,
				map[string]string{"owner": "alice"}, "", ""),
			selector: kptfilev1.Selector{
				Annotations: map[string]string{"owner": "alice"},
			},
			want: true,
		},
		{
			name: "annotation no match",
			node: makeNode(t, "v1", "ConfigMap", "cm", "", nil,
				map[string]string{"owner": "alice"}, "", ""),
			selector: kptfilev1.Selector{
				Annotations: map[string]string{"owner": "bob"},
			},
			want: false,
		},
		{
			name:     "regexp match",
			node:     makeNode(t, "apps/v1", "Deployment", "d", "", nil, nil, "/root", "deploy/app.yaml"),
			selector: kptfilev1.Selector{ResourceFileRegexp: `deploy/.*\.yaml`},
			want:     true,
		},
		{
			name:     "regexp no match",
			node:     makeNode(t, "apps/v1", "Deployment", "d", "", nil, nil, "/root", "other/app.yaml"),
			selector: kptfilev1.Selector{ResourceFileRegexp: `deploy/.*\.yaml`},
			want:     false,
		},
		{
			name:     "invalid regexp returns error",
			node:     makeNode(t, "apps/v1", "Deployment", "d", "", nil, nil, "/root", "deploy.yaml"),
			selector: kptfilev1.Selector{ResourceFileRegexp: `[invalid`},
			wantErr:  true,
		},
		{
			name: "all criteria match",
			node: makeNode(t, "apps/v1", "Deployment", "my-app", "prod",
				map[string]string{"env": "prod"},
				map[string]string{"owner": "team-a"},
				"/root", "deploy/app.yaml"),
			selector: kptfilev1.Selector{
				APIVersion:         "apps/v1",
				Kind:               "Deployment",
				Name:               "my-app",
				Namespace:          "prod",
				Labels:             map[string]string{"env": "prod"},
				Annotations:        map[string]string{"owner": "team-a"},
				ResourceFileRegexp: `deploy/.*`,
			},
			want: true,
		},
		{
			name: "all criteria match except one label",
			node: makeNode(t, "apps/v1", "Deployment", "my-app", "prod",
				map[string]string{"env": "staging"},
				map[string]string{"owner": "team-a"},
				"/root", "deploy/app.yaml"),
			selector: kptfilev1.Selector{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "my-app",
				Namespace:  "prod",
				Labels:     map[string]string{"env": "prod"},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IsMatch(tc.node, tc.selector, ctx)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResourceFileRegexpMatch(t *testing.T) {
	tests := []struct {
		name     string
		pkgPath  string
		filePath string
		regexp   string
		ctx      *SelectionContext
		want     bool
		wantErr  bool
	}{
		{
			name:     "empty regexp always matches",
			pkgPath:  "/root",
			filePath: "deploy.yaml",
			regexp:   "",
			ctx:      &SelectionContext{RootPackagePath: "/root"},
			want:     true,
		},
		{
			name:     "simple filename match",
			pkgPath:  "/root",
			filePath: "deploy.yaml",
			regexp:   `deploy\.yaml`,
			ctx:      &SelectionContext{RootPackagePath: "/root"},
			want:     true,
		},
		{
			name:     "subdirectory match",
			pkgPath:  "/root",
			filePath: "subpkg/deploy.yaml",
			regexp:   `subpkg/.*`,
			ctx:      &SelectionContext{RootPackagePath: "/root"},
			want:     true,
		},
		{
			name:     "no match",
			pkgPath:  "/root",
			filePath: "other.yaml",
			regexp:   `deploy\.yaml`,
			ctx:      &SelectionContext{RootPackagePath: "/root"},
			want:     false,
		},
		{
			name:     "nil context uses empty root path",
			pkgPath:  "",
			filePath: "deploy.yaml",
			regexp:   `deploy\.yaml`,
			ctx:      nil,
			want:     true,
		},
		{
			name:     "invalid regexp returns error",
			pkgPath:  "/root",
			filePath: "deploy.yaml",
			regexp:   `[invalid`,
			ctx:      &SelectionContext{RootPackagePath: "/root"},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node := makeNode(t, "v1", "ConfigMap", "cm", "", nil, nil, tc.pkgPath, tc.filePath)
			selector := kptfilev1.Selector{ResourceFileRegexp: tc.regexp}
			got, err := resourceFileRegexpMatch(node, selector, tc.ctx)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIndividualMatchFunctions(t *testing.T) {
	t.Run("nameMatch", func(t *testing.T) {
		node := makeNode(t, "v1", "ConfigMap", "my-cm", "", nil, nil, "", "")
		match := nameMatch(node, kptfilev1.Selector{Name: "my-cm"})
		assert.True(t, match)

		match = nameMatch(node, kptfilev1.Selector{Name: "other"})
		assert.False(t, match)

		match = nameMatch(node, kptfilev1.Selector{})
		assert.True(t, match)
	})

	t.Run("namespaceMatch", func(t *testing.T) {
		node := makeNode(t, "v1", "ConfigMap", "cm", "prod", nil, nil, "", "")
		match := namespaceMatch(node, kptfilev1.Selector{Namespace: "prod"})
		assert.True(t, match)

		match = namespaceMatch(node, kptfilev1.Selector{Namespace: "staging"})
		assert.False(t, match)

		match = namespaceMatch(node, kptfilev1.Selector{})
		assert.True(t, match)
	})

	t.Run("kindMatch", func(t *testing.T) {
		node := makeNode(t, "apps/v1", "Deployment", "d", "", nil, nil, "", "")
		match := kindMatch(node, kptfilev1.Selector{Kind: "Deployment"})
		assert.True(t, match)

		match = kindMatch(node, kptfilev1.Selector{Kind: "Service"})
		assert.False(t, match)

		match = kindMatch(node, kptfilev1.Selector{})
		assert.True(t, match)
	})

	t.Run("apiVersionMatch", func(t *testing.T) {
		node := makeNode(t, "apps/v1", "Deployment", "d", "", nil, nil, "", "")
		match := apiVersionMatch(node, kptfilev1.Selector{APIVersion: "apps/v1"})
		assert.True(t, match)

		match = apiVersionMatch(node, kptfilev1.Selector{APIVersion: "v1"})
		assert.False(t, match)

		match = apiVersionMatch(node, kptfilev1.Selector{})
		assert.True(t, match)
	})

	t.Run("labelMatch", func(t *testing.T) {
		node := makeNode(t, "v1", "ConfigMap", "cm", "", map[string]string{"env": "prod", "team": "infra"}, nil, "", "")

		match := labelMatch(node, kptfilev1.Selector{Labels: map[string]string{"env": "prod"}})
		assert.True(t, match)

		// both labels present
		match = labelMatch(node, kptfilev1.Selector{Labels: map[string]string{"env": "prod", "team": "infra"}})
		assert.True(t, match)

		// wrong value
		match = labelMatch(node, kptfilev1.Selector{Labels: map[string]string{"env": "staging"}})
		assert.False(t, match)

		// missing key
		match = labelMatch(node, kptfilev1.Selector{Labels: map[string]string{"owner": "alice"}})
		assert.False(t, match)

		// empty selector labels always match
		match = labelMatch(node, kptfilev1.Selector{})
		assert.True(t, match)
	})

	t.Run("annoMatch", func(t *testing.T) {
		node := makeNode(t, "v1", "ConfigMap", "cm", "", nil,
			map[string]string{"owner": "alice", "team": "infra"}, "", "")

		match := annoMatch(node, kptfilev1.Selector{Annotations: map[string]string{"owner": "alice"}})
		assert.True(t, match)

		// both annotations present
		match = annoMatch(node, kptfilev1.Selector{Annotations: map[string]string{"owner": "alice", "team": "infra"}})
		assert.True(t, match)

		// wrong value
		match = annoMatch(node, kptfilev1.Selector{Annotations: map[string]string{"owner": "bob"}})
		assert.False(t, match)

		// missing key
		match = annoMatch(node, kptfilev1.Selector{Annotations: map[string]string{"env": "prod"}})
		assert.False(t, match)

		// empty selector annotations always match
		match = annoMatch(node, kptfilev1.Selector{})
		assert.True(t, match)
	})
}
