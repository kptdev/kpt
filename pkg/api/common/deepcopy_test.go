// Copyright 2026 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestDeepCopyIntoResult(t *testing.T) {
	in := &framework.Result{
		Severity: framework.Info,
		Message:  "message",
		ResourceRef: &yaml.ResourceIdentifier{
			TypeMeta: yaml.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.Version,
				Kind:       "Deployment",
			},
			NameMeta: yaml.NameMeta{
				Name: "test",
			},
		},
		File: &framework.File{
			Path:  "path/to/test.yaml",
			Index: 3,
		},
		Tags: map[string]string{
			"key": "value",
		},
		Field: &framework.Field{
			Path:          ".spec.replicas",
			CurrentValue:  "0",
			ProposedValue: "3",
		},
	}

	testCases := map[string]struct {
		transformFn func(*testing.T, *framework.Result)
	}{
		"everything filled": {},
		"ResourceRef nil": {
			transformFn: func(tt *testing.T, r *framework.Result) {
				orig := r.ResourceRef
				tt.Cleanup(func() {
					r.ResourceRef = orig
				})
				r.ResourceRef = nil
			},
		},
		"File nil": {
			transformFn: func(tt *testing.T, r *framework.Result) {
				orig := r.File
				tt.Cleanup(func() {
					r.File = orig
				})
				r.File = nil
			},
		},
		"Field nil": {
			transformFn: func(tt *testing.T, r *framework.Result) {
				orig := r.Field
				tt.Cleanup(func() {
					r.Field = orig
				})
				r.Field = nil
			},
		},
		"Tags nil": {
			transformFn: func(tt *testing.T, r *framework.Result) {
				orig := r.Tags
				tt.Cleanup(func() {
					r.Tags = orig
				})
				r.Tags = nil
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.transformFn != nil {
				tc.transformFn(t, in)
			}

			out := &framework.Result{}
			require.NotPanics(t, func() {
				DeepCopyIntoResult(in, out)
			})

			assert.Equal(t, in.Severity, out.Severity)
			assert.Equal(t, in.Message, out.Message)
			assert.Equal(t, in.Tags, out.Tags)

			if in.ResourceRef != nil {
				_ = assert.NotNil(t, out.ResourceRef) &&
					assert.Equal(t, *in.ResourceRef, *out.ResourceRef)
			} else {
				assert.Nil(t, out.ResourceRef)
			}

			if in.File != nil {
				_ = assert.NotNil(t, out.File) &&
					assert.Equal(t, *in.File, *out.File)
			} else {
				assert.Nil(t, out.File)
			}

			if in.Field != nil {
				if assert.NotNil(t, out.Field) {
					assert.Equal(t, in.Field.Path, out.Field.Path)
					assert.Equal(t, in.Field.CurrentValue, out.Field.CurrentValue)
					assert.Equal(t, in.Field.ProposedValue, out.Field.ProposedValue)
				}
			} else {
				assert.Nil(t, out.Field)
			}
		})
	}

	t.Run("nil", func(t *testing.T) {
		out := &framework.Result{}
		assert.NotPanics(t, func() {
			DeepCopyIntoResult(nil, out)
		})
		assert.NotNil(t, out)
	})
}

func TestDeepCopyIntoResultFieldValues(t *testing.T) {
	testCases := map[string]struct {
		value any
	}{
		"nil": {
			value: nil,
		},
		"int": {
			value: 3,
		},
		"float": {
			value: 3.14,
		},
		"string": {
			value: "a",
		},
		"slice": {
			value: []any{"a", "b", "c"},
		},
		"map": {
			value: map[string]any{
				"a": "b",
				"c": "d",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			in := &framework.Result{
				Field: &framework.Field{
					CurrentValue:  tc.value,
					ProposedValue: tc.value, // for coverage
				},
			}

			out := &framework.Result{}
			require.NotPanics(t, func() {
				DeepCopyIntoResult(in, out)
			})

			assert.Equal(t, in.Field.CurrentValue, out.Field.CurrentValue)
			assert.Equal(t, in.Field.ProposedValue, out.Field.ProposedValue)
		})
	}
}

func TestDeepCopyIntoResults(t *testing.T) {
	testCases := map[string]struct {
		in *framework.Results
	}{
		"nil": {
			in: nil,
		},
		"empty": {
			in: &framework.Results{},
		},
		"few": {
			in: &framework.Results{
				{
					Severity: framework.Error,
					Message:  "message",
				},
				{
					Severity: framework.Info,
					Message:  "message2",
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			out := &framework.Results{}
			require.NotPanics(t, func() {
				DeepCopyIntoResults(tc.in, out)
			})

			expectedLen := 0
			if tc.in != nil {
				expectedLen = len(*tc.in)
			}
			assert.Len(t, *out, expectedLen)
		})
	}
}

func TestDeepCopyInterfacePanics(t *testing.T) {
	t.Run("unhandled type", func(t *testing.T) {
		assert.PanicsWithValue(t, "cannot deepcopy type map[string]string", func() {
			var in any = map[string]string{
				"a": "b",
			}

			_ = DeepCopyInterface(in)
		})
	})

	t.Run("max depth", func(t *testing.T) {
		assert.PanicsWithValue(t, "reached max deepcopy depth of 1024", func() {
			var in = map[string]any{}
			current := &in

			for range 1025 {
				next := make(map[string]any)
				(*current)["a"] = next
				current = &next
			}

			_ = DeepCopyInterface(in)
		})
	})
}
