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

package live

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestValidateAnnotationTypes is a Table-Driven Test (TDT) suite for the
// validateAnnotationTypes function. It verifies that:
// - String annotations (including quoted "true", "123") are ACCEPTED
// - Raw non-string values (true, 123, 1.5) are REJECTED with clear errors
// - Empty/missing annotations are handled without panics
func TestValidateAnnotationTypes(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name: "valid_string",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: bar
`,
			expectError: false,
		},
		{
			name: "quoted_boolean",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: "true"
`,
			expectError: false,
		},
		{
			name: "quoted_number",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: "123"
`,
			expectError: false,
		},
		{
			name: "empty_annotations",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations: {}
`,
			expectError: false,
		},
		{
			name: "no_annotations_field",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`,
			expectError: false,
		},
		{
			name: "no_metadata",
			input: `
apiVersion: v1
kind: ConfigMap
`,
			expectError: false,
		},
		{
			name: "trap_boolean",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: true
`,
			expectError: true,
		},
		{
			name: "trap_integer",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: 123
`,
			expectError: true,
		},
		{
			name: "trap_float",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: 1.5
`,
			expectError: true,
		},
		{
			name: "trap_null",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: null
`,
			expectError: true,
		},
		{
			name: "trap_boolean_false",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: false
`,
			expectError: true,
		},
		{
			name: "trap_scientific_notation",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: 1e10
`,
			expectError: true,
		},
		{
			name: "mixed_valid_and_invalid",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    good: "valid-string"
    bad: true
`,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the YAML input into an RNode
			node, err := yaml.Parse(tc.input)
			require.NoError(t, err, "failed to parse test YAML")

			// Run the validator
			err = validateAnnotationTypes(node)

			// Assert the expected outcome
			if tc.expectError {
				assert.Error(t, err, "expected validation to fail for %s", tc.name)
			} else {
				assert.NoError(t, err, "expected validation to pass for %s", tc.name)
			}
		})
	}
}

// TestValidateAnnotationTypes_ErrorMessages verifies that error messages
// are clear and identify the problematic annotation key.
func TestValidateAnnotationTypes_ErrorMessages(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name: "boolean_error_message",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    my-annotation: true
`,
			errContains: "my-annotation",
		},
		{
			name: "integer_error_message",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    replica-count: 3
`,
			errContains: "replica-count",
		},
		{
			name: "boolean_type_in_message",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    enabled: true
`,
			errContains: "boolean",
		},
		{
			name: "integer_type_in_message",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    count: 42
`,
			errContains: "integer",
		},
		{
			name: "float_type_in_message",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    ratio: 3.14
`,
			errContains: "number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := yaml.Parse(tc.input)
			require.NoError(t, err)

			err = validateAnnotationTypes(node)
			require.Error(t, err, "expected validation to fail")
			assert.Contains(t, err.Error(), tc.errContains,
				"error message should contain %q", tc.errContains)
		})
	}
}

// TestValidateAnnotationTypes_NullAnnotationsField verifies that
// an explicit `annotations: null` is handled gracefully (not an error).
func TestValidateAnnotationTypes_NullAnnotationsField(t *testing.T) {
	input := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations: null
`
	node, err := yaml.Parse(input)
	require.NoError(t, err)

	err = validateAnnotationTypes(node)
	assert.NoError(t, err, "annotations: null should be accepted")
}

// TestValidateAnnotationTypes_NonScalarValue verifies that
// non-scalar annotation values (maps, sequences) are rejected.
func TestValidateAnnotationTypes_NonScalarValue(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name: "map_value",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    nested:
      key: value
`,
		},
		{
			name: "sequence_value",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    list:
      - item1
      - item2
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := yaml.Parse(tc.input)
			require.NoError(t, err)

			err = validateAnnotationTypes(node)
			assert.Error(t, err, "non-scalar annotation value should be rejected")
			assert.Contains(t, err.Error(), "non-scalar")
		})
	}
}

// TestYamlTagToType verifies the helper function that converts
// YAML tags to human-readable type names.
func TestYamlTagToType(t *testing.T) {
	testCases := []struct {
		tag      string
		kind     yaml.Kind
		expected string
	}{
		{tag: "!!bool", kind: yaml.ScalarNode, expected: "boolean"},
		{tag: "!!int", kind: yaml.ScalarNode, expected: "integer"},
		{tag: "!!float", kind: yaml.ScalarNode, expected: "number"},
		{tag: "!!null", kind: yaml.ScalarNode, expected: "null"},
		{tag: "!!str", kind: yaml.ScalarNode, expected: "string"},
		{tag: "!!custom", kind: yaml.ScalarNode, expected: "!!custom"},
		{tag: "", kind: yaml.ScalarNode, expected: "unknown"},
		{tag: "!!str", kind: yaml.MappingNode, expected: "non-scalar"},
		{tag: "!!str", kind: yaml.SequenceNode, expected: "non-scalar"},
	}

	for _, tc := range testCases {
		t.Run(tc.tag+"_"+tc.expected, func(t *testing.T) {
			node := &yaml.Node{
				Kind: tc.kind,
				Tag:  tc.tag,
			}
			result := yamlTagToType(node)
			assert.Equal(t, tc.expected, result)
		})
	}
}
