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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	kyamlmerge3 "sigs.k8s.io/kustomize/kyaml/yaml/merge3"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

// twoFieldDoc is a minimal mapping. key1 is the field under test; key2 stays
// put so a cleared key1 does not collapse the document to empty.
func twoFieldDoc(key1 string) string {
	return fmt.Sprintf("key1: %s\nkey2: control\n", key1)
}

func twoFieldDocNoKey1() string {
	return "key2: control\n"
}

func deploymentWithImage(image string) string {
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      containers:
      - name: web
        image: %s
`, image)
}

const deploymentWithNullContainers = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      containers: null
`

func mergeWithVisitor(t *testing.T, visitor walk.Visitor, origin, updated, dest string) string {
	t.Helper()
	openapi.ResetOpenAPI()
	o, err := yaml.Parse(origin)
	require.NoError(t, err)
	u, err := yaml.Parse(updated)
	require.NoError(t, err)
	d, err := yaml.Parse(dest)
	require.NoError(t, err)

	// Same walker settings as tuple.merge and kyaml merge3.Merge.
	result, err := walk.Walker{
		Visitor:               visitor,
		VisitKeysAsScalars:    true,
		InferAssociativeLists: false,
		Sources:               []*yaml.RNode{d, o, u},
	}.Walk()
	require.NoError(t, err)
	if result == nil {
		return ""
	}
	out, err := result.String()
	require.NoError(t, err)
	return out
}

// TestVisitorMatchesKyaml covers merges where kyaml, Visitor, and
// NullPreservingVisitor all agree.
func TestVisitorMatchesKyaml(t *testing.T) {
	testCases := map[string]struct {
		origin, updated, dest string
	}{
		"unchanged scalar": {
			origin:  twoFieldDoc("a"),
			updated: twoFieldDoc("a"),
			dest:    twoFieldDoc("a"),
		},
		"updated scalar wins when dest is unchanged": {
			origin:  twoFieldDoc("a"),
			updated: twoFieldDoc("b"),
			dest:    twoFieldDoc("a"),
		},
		"dest scalar kept when update is unchanged": {
			origin:  twoFieldDoc("a"),
			updated: twoFieldDoc("a"),
			dest:    twoFieldDoc("local"),
		},
		"updated scalar wins when both sides changed": {
			origin:  twoFieldDoc("a"),
			updated: twoFieldDoc("b"),
			dest:    twoFieldDoc("local"),
		},
		"key added by update": {
			origin:  twoFieldDocNoKey1(),
			updated: twoFieldDoc("added"),
			dest:    twoFieldDocNoKey1(),
		},
		"key removed by update": {
			origin:  twoFieldDoc("a"),
			updated: twoFieldDocNoKey1(),
			dest:    twoFieldDoc("a"),
		},
		"key removed by dest": {
			origin:  twoFieldDoc("a"),
			updated: twoFieldDoc("a"),
			dest:    twoFieldDocNoKey1(),
		},
		"non-associative list updated when content changes": {
			origin:  "key1:\n- a\nkey2: control\n",
			updated: "key1:\n- b\nkey2: control\n",
			dest:    "key1:\n- a\nkey2: control\n",
		},
		"non-associative list dest kept when update is unchanged": {
			origin:  "key1:\n- a\nkey2: control\n",
			updated: "key1:\n- a\nkey2: control\n",
			dest:    "key1:\n- local\nkey2: control\n",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			plain := mergeWithVisitor(t, &Visitor{}, tc.origin, tc.updated, tc.dest)
			preserving := mergeWithVisitor(t, &NullPreservingVisitor{}, tc.origin, tc.updated, tc.dest)
			kyaml := mergeWithVisitor(t, kyamlmerge3.Visitor{}, tc.origin, tc.updated, tc.dest)
			assert.Equal(t, kyaml, plain)
			assert.Equal(t, plain, preserving)
		})
	}
}

// TestVisitorDiffersFromKyaml covers merges where Visitor and NullPreservingVisitor
// agree with each other and differ from kyaml. A null only deletes a field when
// origin was not already null. kyaml deletes whenever updated or dest is tagged null.
func TestVisitorDiffersFromKyaml(t *testing.T) {
	testCases := map[string]struct {
		origin, updated, dest string
		wantPlain, wantKyaml  string
	}{
		"updated value replaces origin null": {
			origin:    twoFieldDoc("null"),
			updated:   twoFieldDoc("new"),
			dest:      twoFieldDoc("null"),
			wantPlain: twoFieldDoc("new"),
			wantKyaml: twoFieldDocNoKey1(),
		},
		"dest value kept when origin and update are null": {
			origin:    twoFieldDoc("null"),
			updated:   twoFieldDoc("null"),
			dest:      twoFieldDoc("local"),
			wantPlain: twoFieldDoc("local"),
			wantKyaml: twoFieldDocNoKey1(),
		},
		"updated list replaces origin null": {
			origin:    twoFieldDoc("null"),
			updated:   "key1:\n- b\nkey2: control\n",
			dest:      twoFieldDoc("null"),
			wantPlain: "key1:\n- b\nkey2: control\n",
			wantKyaml: twoFieldDocNoKey1(),
		},
		// Direct walk, no dropOriginUpdatedAtDestNulls, so both visitors delete the list.
		// Merge(preserve=true) keeps it; see TestPreserveExplicitNullAssociativeList.
		"dest null removes an associative list origin had": {
			origin:  deploymentWithImage("nginx"),
			updated: deploymentWithImage("nginx:updated"),
			dest:    deploymentWithNullContainers,
			wantPlain: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec: {}
`,
			wantKyaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      containers:
      - image: nginx:updated
        name: web
`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			plain := mergeWithVisitor(t, &Visitor{}, tc.origin, tc.updated, tc.dest)
			preserving := mergeWithVisitor(t, &NullPreservingVisitor{}, tc.origin, tc.updated, tc.dest)
			kyaml := mergeWithVisitor(t, kyamlmerge3.Visitor{}, tc.origin, tc.updated, tc.dest)
			assert.Equal(t, tc.wantPlain, plain)
			assert.Equal(t, plain, preserving)
			assert.Equal(t, tc.wantKyaml, kyaml)
			assert.NotEqual(t, kyaml, plain)
		})
	}
}

// TestNullPreservingVisitorDiffersFromVisitor covers tagged nulls. Visitor and
// kyaml delete them; NullPreservingVisitor keeps the null so a patch can stay explicit.
func TestNullPreservingVisitorDiffersFromVisitor(t *testing.T) {
	implicitNull := "key1:\nkey2: control\n"
	testCases := map[string]struct {
		origin, updated, dest string
		wantPlain             string
		wantPreserving        string
	}{
		"updated explicit null is kept": {
			origin:         twoFieldDoc("a"),
			updated:        twoFieldDoc("null"),
			dest:           twoFieldDoc("local"),
			wantPlain:      twoFieldDocNoKey1(),
			wantPreserving: twoFieldDoc("null"),
		},
		"dest explicit null is kept": {
			origin:         twoFieldDoc("a"),
			updated:        twoFieldDoc("b"),
			dest:           twoFieldDoc("null"),
			wantPlain:      twoFieldDocNoKey1(),
			wantPreserving: twoFieldDoc("null"),
		},
		"explicit null on every side is kept": {
			origin:         twoFieldDoc("null"),
			updated:        twoFieldDoc("null"),
			dest:           twoFieldDoc("null"),
			wantPlain:      twoFieldDocNoKey1(),
			wantPreserving: twoFieldDoc("null"),
		},
		"updated implicit null is kept": {
			origin:         twoFieldDoc("a"),
			updated:        implicitNull,
			dest:           twoFieldDoc("local"),
			wantPlain:      twoFieldDocNoKey1(),
			wantPreserving: implicitNull,
		},
		"dest implicit null is kept": {
			origin:         twoFieldDoc("a"),
			updated:        twoFieldDoc("b"),
			dest:           implicitNull,
			wantPlain:      twoFieldDocNoKey1(),
			wantPreserving: implicitNull,
		},
		"updated null list is kept": {
			origin:         "key1:\n- a\nkey2: control\n",
			updated:        twoFieldDoc("null"),
			dest:           "key1:\n- local\nkey2: control\n",
			wantPlain:      twoFieldDocNoKey1(),
			wantPreserving: twoFieldDoc("null"),
		},
		"updated map replaces origin null": {
			origin:         twoFieldDoc("null"),
			updated:        "key1:\n  nested: value\nkey2: control\n",
			dest:           twoFieldDoc("null"),
			wantPlain:      twoFieldDocNoKey1(),
			wantPreserving: "key1:\n  nested: value\nkey2: control\n",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			plain := mergeWithVisitor(t, &Visitor{}, tc.origin, tc.updated, tc.dest)
			preserving := mergeWithVisitor(t, &NullPreservingVisitor{}, tc.origin, tc.updated, tc.dest)
			kyaml := mergeWithVisitor(t, kyamlmerge3.Visitor{}, tc.origin, tc.updated, tc.dest)
			assert.Equal(t, tc.wantPlain, plain)
			assert.Equal(t, plain, kyaml)
			assert.Equal(t, tc.wantPreserving, preserving)
			assert.NotEqual(t, plain, preserving)
		})
	}
}
