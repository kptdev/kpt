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
//
// Portions of this file are adapted from the Kubernetes apimachinery project:
// https://github.com/kubernetes/apimachinery/blob/v0.34.9/pkg/runtime/schema/group_version.go
//
// Copyright 2015 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGroupVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    GroupVersion
		wantErr bool
	}{
		{name: "core version", input: "v1", want: GroupVersion{Version: "v1"}},
		{name: "grouped version", input: "apps/v1", want: GroupVersion{Group: "apps", Version: "v1"}},
		{name: "empty", input: "", want: GroupVersion{}},
		{name: "slash only", input: "/", want: GroupVersion{}},
		{name: "invalid", input: "a/b/c", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseGroupVersion(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGroupVersionKindGroupKind(t *testing.T) {
	t.Parallel()

	gvk := GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	assert.Equal(t, GroupKind{Group: "apps", Kind: "Deployment"}, gvk.GroupKind())
	assert.Equal(t, GroupVersion{Group: "apps", Version: "v1"}, gvk.GroupVersion())
	assert.Equal(t, "apps/v1, Kind=Deployment", gvk.String())
}

func TestGroupKindString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ConfigMap", GroupKind{Kind: "ConfigMap"}.String())
	assert.Equal(t, "SetLabelsFn.kpt.dev", GroupKind{Group: "kpt.dev", Kind: "SetLabelsFn"}.String())
}

func TestGroupVersionString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "v1", GroupVersion{Version: "v1"}.String())
	assert.Equal(t, "apps/v1", GroupVersion{Group: "apps", Version: "v1"}.String())
}

func TestGroupVersionWithKind(t *testing.T) {
	t.Parallel()

	gv, err := ParseGroupVersion("apps/v2")
	require.NoError(t, err)
	assert.Equal(t, GroupVersionKind{Group: "apps", Version: "v2", Kind: "StatefulSet"}, gv.WithKind("StatefulSet"))
}
