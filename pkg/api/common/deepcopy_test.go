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
}

func TestDeepCopyIntoResultCurrentValue(t *testing.T) {
	testCases := map[string]struct {
		value any
	}{
		"int CurrentValue": {
			value: 3,
		},
		"float CurrentValue": {
			value: 3.14,
		},
		"string CurrentValue": {
			value: "a",
		},
		"slice CurrentValue": {
			value: []any{"a", "b", "c"},
		},
		"map CurrentValue": {
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
					CurrentValue: tc.value,
				},
			}

			out := &framework.Result{}
			require.NotPanics(t, func() {
				DeepCopyIntoResult(in, out)
			})

			assert.Equal(t, in.Field.CurrentValue, out.Field.CurrentValue)
		})
	}
}
