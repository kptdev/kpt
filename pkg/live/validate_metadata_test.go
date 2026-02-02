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

// TestValidateMetadataStringMaps is a Table-Driven Test (TDT) suite for the
// validateMetadataStringMaps function. It verifies that:
// - String annotations and labels (including quoted "true", "123") are ACCEPTED
// - Raw non-string values (true, 123, 1.5) are REJECTED with clear errors
// - Empty/missing metadata fields are handled without panics
func TestValidateMetadataStringMaps(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectError bool
		errContains string
	}{
		// ========== ANNOTATIONS TESTS ==========
		{
			name: "valid_string_annotation",
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
			name: "quoted_boolean_annotation",
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
			name: "quoted_number_annotation",
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
			name: "null_annotations",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations: null
`,
			expectError: false,
		},
		{
			name: "trap_boolean_annotation",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: true
`,
			expectError: true,
			errContains: "annotation",
		},
		{
			name: "trap_integer_annotation",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: 123
`,
			expectError: true,
			errContains: "annotation",
		},
		{
			name: "trap_float_annotation",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    foo: 1.5
`,
			expectError: true,
			errContains: "annotation",
		},

		// ========== LABELS TESTS ==========
		{
			name: "valid_string_label",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    app: myapp
`,
			expectError: false,
		},
		{
			name: "quoted_boolean_label",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    enabled: "true"
`,
			expectError: false,
		},
		{
			name: "quoted_number_label",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    version: "123"
`,
			expectError: false,
		},
		{
			name: "empty_labels",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels: {}
`,
			expectError: false,
		},
		{
			name: "null_labels",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels: null
`,
			expectError: false,
		},
		{
			name: "trap_boolean_label",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    enabled: true
`,
			expectError: true,
			errContains: "label",
		},
		{
			name: "trap_integer_label",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    replica-count: 3
`,
			expectError: true,
			errContains: "label",
		},
		{
			name: "trap_float_label",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    ratio: 1.5
`,
			expectError: true,
			errContains: "label",
		},

		// ========== EDGE CASES ==========
		{
			name: "no_metadata",
			input: `
apiVersion: v1
kind: ConfigMap
`,
			expectError: false,
		},
		{
			name: "metadata_without_annotations_or_labels",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`,
			expectError: false,
		},
		{
			name: "valid_annotations_and_labels",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    app: myapp
  annotations:
    description: "my config"
`,
			expectError: false,
		},
		{
			name: "valid_annotations_invalid_labels",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    enabled: true
  annotations:
    description: "valid"
`,
			expectError: true,
			errContains: "label",
		},
		{
			name: "invalid_annotations_valid_labels",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    app: myapp
  annotations:
    enabled: true
`,
			expectError: true,
			errContains: "annotation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := yaml.Parse(tc.input)
			require.NoError(t, err, "failed to parse test YAML")

			err = validateMetadataStringMaps(node)

			if tc.expectError {
				require.Error(t, err, "expected validation to fail for %s", tc.name)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains,
						"error should mention %q", tc.errContains)
				}
			} else {
				assert.NoError(t, err, "expected validation to pass for %s", tc.name)
			}
		})
	}
}

// TestValidateMetadataStringMaps_ErrorMessages verifies that error messages
// clearly identify which field (annotation vs label) and which key failed.
func TestValidateMetadataStringMaps_ErrorMessages(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		errContains []string
	}{
		{
			name: "annotation_error_includes_key_and_type",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    my-annotation: true
`,
			errContains: []string{"annotation", "my-annotation", "boolean"},
		},
		{
			name: "label_error_includes_key_and_type",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    my-label: 123
`,
			errContains: []string{"label", "my-label", "integer"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := yaml.Parse(tc.input)
			require.NoError(t, err)

			err = validateMetadataStringMaps(node)
			require.Error(t, err, "expected validation to fail")

			for _, substr := range tc.errContains {
				assert.Contains(t, err.Error(), substr,
					"error message should contain %q", substr)
			}
		})
	}
}

// TestValidateStringMap_NonScalarValues verifies that non-scalar values
// (maps, sequences) are rejected for both annotations and labels.
func TestValidateStringMap_NonScalarValues(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name: "annotation_map_value",
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
			name: "annotation_sequence_value",
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
		{
			name: "label_map_value",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    nested:
      key: value
`,
		},
		{
			name: "label_sequence_value",
			input: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
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

			err = validateMetadataStringMaps(node)
			assert.Error(t, err, "non-scalar value should be rejected")
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
