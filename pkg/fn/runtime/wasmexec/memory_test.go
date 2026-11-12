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

package wasmexec

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetArgs(t *testing.T) {
	instance := newTestInstance()

	argc, argv, err := SetArgs(instance, []string{"prog", "arg"}, []string{"KEY=value"})
	require.NoError(t, err)
	assert.Equal(t, int32(2), argc)

	// argv points at a list of pointers: args, nil, envs, nil.
	readString := func(ptr uint32) string {
		end := ptr
		for instance.mem[end] != 0 {
			end++
		}
		return string(instance.mem[ptr:end])
	}
	readPointer := func(index uint32) uint32 {
		ptr, err := instance.GetUInt32(uint32(argv) + index*8)
		require.NoError(t, err)
		return ptr
	}

	assert.Equal(t, "prog", readString(readPointer(0)))
	assert.Equal(t, "arg", readString(readPointer(1)))
	assert.Equal(t, uint32(0), readPointer(2), "args are terminated by a nil pointer")
	assert.Equal(t, "KEY=value", readString(readPointer(3)))
	assert.Equal(t, uint32(0), readPointer(4), "envs are terminated by a nil pointer")
}

func TestSetArgsNoArgs(t *testing.T) {
	argc, argv, err := SetArgs(newTestInstance(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int32(0), argc)
	assert.Positive(t, argv)
}

func TestSetArgsExceedsLimit(t *testing.T) {
	_, _, err := SetArgs(newTestInstance(), []string{strings.Repeat("a", wasmMinDataAddr)}, nil)
	assert.Error(t, err)
}

func TestSetArgsOutOfMemory(t *testing.T) {
	instance := &testInstance{mem: make([]byte, 16)}

	_, _, err := SetArgs(instance, []string{"prog"}, nil)
	assert.ErrorIs(t, err, ErrFault)
}
