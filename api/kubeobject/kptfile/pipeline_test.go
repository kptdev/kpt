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

package kptfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kptfileapi "github.com/kptdev/kpt/api/kptfile/v1"
)

func TestUpsertMutatorFunctions(t *testing.T) {
	kf := newEmptyKptfile(t)
	fn := kptfileapi.Function{
		Name:  "set-image",
		Image: "set-image:v1",
	}

	err := kf.UpsertMutatorFunctions([]kptfileapi.Function{fn}, 0)
	require.NoError(t, err)

	mutators, _, err := kf.UpsertMap("pipeline").NestedSlice("mutators")
	require.NoError(t, err)
	require.Len(t, mutators, 1)

	var gotFn kptfileapi.Function
	require.NoError(t, mutators[0].As(&gotFn))
	assert.Equal(t, fn.Name, gotFn.Name)
	assert.Equal(t, fn.Image, gotFn.Image)
}

func TestUpsertValidatorFunctions(t *testing.T) {
	kf := newEmptyKptfile(t)
	fn1 := kptfileapi.Function{
		Name:  "gatekeeper",
		Image: "gatekeeper:v1",
	}

	err := kf.UpsertValidatorFunctions([]kptfileapi.Function{fn1}, 0)
	require.NoError(t, err)

	validators, _, err := kf.UpsertMap("pipeline").NestedSlice("validators")
	require.NoError(t, err)
	require.Len(t, validators, 1)

	var gotFn kptfileapi.Function
	require.NoError(t, validators[0].As(&gotFn))
	assert.Equal(t, fn1.Name, gotFn.Name)
	assert.Equal(t, fn1.Image, gotFn.Image)
}

func TestUpsertPipelineFunctions(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		kf := newEmptyKptfile(t)
		fn := kptfileapi.Function{
			Name:  "set-image",
			Image: "set-image:v1",
		}

		err := kf.UpsertPipelineFunctions([]kptfileapi.Function{fn}, "mutators", 0)
		require.NoError(t, err)
		mutators, _, err := kf.UpsertMap("pipeline").NestedSlice("mutators")
		require.NoError(t, err)
		require.Len(t, mutators, 1)

		var gotFn kptfileapi.Function
		require.NoError(t, mutators[0].As(&gotFn))
		assert.Equal(t, fn.Name, gotFn.Name)
		assert.Equal(t, fn.Image, gotFn.Image)
	})

	t.Run("Append", func(t *testing.T) {
		kf := newEmptyKptfile(t)
		fn1 := kptfileapi.Function{
			Name:  "set-image",
			Image: "set-image:v1",
		}
		fn2 := kptfileapi.Function{
			Name:  "set-labels",
			Image: "set-labels:v1",
		}

		err := kf.UpsertPipelineFunctions([]kptfileapi.Function{fn1}, "mutators", 0)
		require.NoError(t, err)

		err = kf.UpsertPipelineFunctions([]kptfileapi.Function{fn2}, "mutators", -1)
		require.NoError(t, err)
		mutators, _, err := kf.UpsertMap("pipeline").NestedSlice("mutators")
		require.NoError(t, err)
		require.Len(t, mutators, 2)

		var gotFn kptfileapi.Function
		require.NoError(t, mutators[1].As(&gotFn))
		assert.Equal(t, fn2.Name, gotFn.Name)
		assert.Equal(t, fn2.Image, gotFn.Image)
	})

	t.Run("Update", func(t *testing.T) {
		kf := newEmptyKptfile(t)
		fn1 := kptfileapi.Function{
			Name:  "set-image",
			Image: "set-image:v1",
		}
		fn1Updated := kptfileapi.Function{
			Name:  "set-image",
			Image: "set-image:v2",
		}

		err := kf.UpsertPipelineFunctions([]kptfileapi.Function{fn1}, "mutators", 0)
		require.NoError(t, err)

		err = kf.UpsertPipelineFunctions([]kptfileapi.Function{fn1Updated}, "mutators", 0)
		require.NoError(t, err)
		mutators, _, err := kf.UpsertMap("pipeline").NestedSlice("mutators")
		require.NoError(t, err)
		require.Len(t, mutators, 1)

		var gotFn kptfileapi.Function
		require.NoError(t, mutators[0].As(&gotFn))
		assert.Equal(t, fn1Updated.Name, gotFn.Name)
		assert.Equal(t, fn1Updated.Image, gotFn.Image)
	})

	t.Run("No duplicate same content", func(t *testing.T) {
		kf := newEmptyKptfile(t)
		fn1 := kptfileapi.Function{
			Name:  "set-image",
			Image: "set-image:v1",
		}
		fn1Updated := kptfileapi.Function{
			Image: "set-image:v1",
		}

		err := kf.UpsertPipelineFunctions([]kptfileapi.Function{fn1}, "mutators", 0)
		require.NoError(t, err)

		err = kf.UpsertPipelineFunctions([]kptfileapi.Function{fn1Updated}, "mutators", 0)
		require.NoError(t, err)
		mutators, _, err := kf.UpsertMap("pipeline").NestedSlice("mutators")
		require.NoError(t, err)
		require.Len(t, mutators, 1)

		var gotFn kptfileapi.Function
		require.NoError(t, mutators[0].As(&gotFn))
		assert.Equal(t, fn1.Name, gotFn.Name)
		assert.Equal(t, fn1.Image, gotFn.Image)
	})

	t.Run("Empty input", func(t *testing.T) {
		kf := newEmptyKptfile(t)
		err := kf.UpsertPipelineFunctions([]kptfileapi.Function{}, "mutators", 0)
		require.NoError(t, err)
		mutators, _, _ := kf.UpsertMap("pipeline").NestedSlice("mutators")
		assert.Len(t, mutators, 0)
	})
}
