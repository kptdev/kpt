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
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testMemorySize = 64 * 1024

// testInstance is an Instance that is backed by a plain byte slice.
type testInstance struct {
	mem []byte
	sp  uint32
}

func newTestInstance() *testInstance {
	return &testInstance{mem: make([]byte, testMemorySize)}
}

var _ Instance = &testInstance{}

func (i *testInstance) check(offset, length uint32) error {
	if uint64(offset)+uint64(length) > uint64(len(i.mem)) {
		return ErrFault
	}
	return nil
}

func (i *testInstance) Range(offset, length uint32) ([]byte, error) {
	if err := i.check(offset, length); err != nil {
		return nil, err
	}
	return i.mem[offset : offset+length], nil
}

func (i *testInstance) GetUInt32(offset uint32) (uint32, error) {
	if err := i.check(offset, 4); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(i.mem[offset:]), nil
}

func (i *testInstance) GetInt64(offset uint32) (int64, error) {
	if err := i.check(offset, 8); err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(i.mem[offset:])), nil
}

func (i *testInstance) GetFloat64(offset uint32) (float64, error) {
	if err := i.check(offset, 8); err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(i.mem[offset:])), nil
}

func (i *testInstance) SetUInt8(offset uint32, val uint8) error {
	if err := i.check(offset, 1); err != nil {
		return err
	}
	i.mem[offset] = val
	return nil
}

func (i *testInstance) SetUInt32(offset, val uint32) error {
	if err := i.check(offset, 4); err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(i.mem[offset:], val)
	return nil
}

func (i *testInstance) SetInt64(offset uint32, val int64) error {
	if err := i.check(offset, 8); err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(i.mem[offset:], uint64(val))
	return nil
}

func (i *testInstance) SetFloat64(offset uint32, val float64) error {
	if err := i.check(offset, 8); err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(i.mem[offset:], math.Float64bits(val))
	return nil
}

func (i *testInstance) GetSP() (uint32, error) { return i.sp, nil }

func (i *testInstance) Resume() error { return nil }

func TestStoreAndLoadValue(t *testing.T) {
	tests := map[string]struct {
		value    any
		expected any
	}{
		"number":    {value: float64(42.5), expected: float64(42.5)},
		"integer":   {value: 7, expected: float64(7)},
		"zero":      {value: float64(0), expected: float64(0)},
		"null":      {value: nil, expected: nil},
		"true":      {value: true, expected: true},
		"false":     {value: false, expected: false},
		"undefined": {value: undefined, expected: nil},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mod := New(newTestInstance())

			require.NoError(t, mod.storeValue(128, tc.value))
			got, err := mod.loadValue(128)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestStoreAndLoadObjectValues(t *testing.T) {
	mod := New(newTestInstance())

	global := mod.values[5]
	require.NoError(t, mod.storeValue(128, global))
	got, err := mod.loadValue(128)
	require.NoError(t, err)
	assert.Same(t, global, got)

	require.NoError(t, mod.storeValue(256, "hello"))
	got, err = mod.loadValue(256)
	require.NoError(t, err)
	assert.Equal(t, &jsString{data: "hello"}, got)
}

func TestStoreUndefinedWritesZeroFloat(t *testing.T) {
	instance := newTestInstance()
	mod := New(instance)

	// Pre-fill the slot to make sure that it is overwritten.
	require.NoError(t, instance.SetInt64(128, -1))
	require.NoError(t, mod.storeValue(128, undefined))

	f, err := instance.GetFloat64(128)
	require.NoError(t, err)
	assert.Equal(t, float64(0), f)
}

func TestStoreNullWritesNaNReference(t *testing.T) {
	instance := newTestInstance()
	mod := New(instance)

	require.NoError(t, mod.storeValue(128, nil))

	f, err := instance.GetFloat64(128)
	require.NoError(t, err)
	assert.True(t, math.IsNaN(f))
	id, err := instance.GetUInt32(128)
	require.NoError(t, err)
	assert.Equal(t, uint32(2), id)
}

func TestReflectGet(t *testing.T) {
	mod := New(newTestInstance())

	t.Run("existing property of the global object", func(t *testing.T) {
		got, err := mod.reflectGet(nil, "Object")
		require.NoError(t, err)
		assert.IsType(t, &jsFunction{}, got)
	})

	t.Run("missing property is undefined, not null", func(t *testing.T) {
		got, err := mod.reflectGet(nil, "doesNotExist")
		require.NoError(t, err)
		assert.Equal(t, undefined, got)
	})

	t.Run("missing property on a properties map is undefined", func(t *testing.T) {
		fsObject, err := mod.reflectGet(nil, "fs")
		require.NoError(t, err)
		constants, err := mod.reflectGet(fsObject, "constants")
		require.NoError(t, err)

		got, err := mod.reflectGet(constants, "O_DIRECTORY")
		require.NoError(t, err)
		assert.Equal(t, undefined, got)
	})

	t.Run("property with a null value stays null", func(t *testing.T) {
		got, err := mod.reflectGet(mod.values[6], "_pendingEvent")
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("array index", func(t *testing.T) {
		got, err := mod.reflectGet(&jsArray{elements: []any{"a", "b"}}, int64(1))
		require.NoError(t, err)
		assert.Equal(t, "b", got)
	})

	t.Run("array index out of range", func(t *testing.T) {
		_, err := mod.reflectGet(&jsArray{elements: []any{"a"}}, int64(1))
		assert.Error(t, err)
	})

	t.Run("string key on a non-object", func(t *testing.T) {
		_, err := mod.reflectGet(&jsArray{}, "length")
		assert.Error(t, err)
	})
}

// TestValueGetMissingPropertyIsUndefined checks the exact scenario that
// Go's syscall package relies on while initializing: reading the missing
// fs.constants.O_DIRECTORY property must yield "undefined" in the guest.
func TestValueGetMissingPropertyIsUndefined(t *testing.T) {
	const (
		sp         = 1024
		nameOffset = 4096
		name       = "O_DIRECTORY"
	)

	instance := newTestInstance()
	instance.sp = sp
	mod := New(instance)

	fsObject, err := mod.reflectGet(nil, "fs")
	require.NoError(t, err)
	constants, err := mod.reflectGet(fsObject, "constants")
	require.NoError(t, err)

	// Lay out the arguments like the guest does.
	require.NoError(t, mod.storeValue(sp+8, constants))
	copy(instance.mem[nameOffset:], name)
	require.NoError(t, instance.SetInt64(sp+16, nameOffset))
	require.NoError(t, instance.SetInt64(sp+24, int64(len(name))))

	mod.ValueGet(sp)

	result, err := instance.GetFloat64(sp + 32)
	require.NoError(t, err)
	assert.Equal(t, float64(0), result, "a zero float64 is how the guest reads undefined")
}

func TestValueGetKnownConstant(t *testing.T) {
	const (
		sp         = 1024
		nameOffset = 4096
		name       = "O_WRONLY"
	)

	instance := newTestInstance()
	instance.sp = sp
	mod := New(instance)

	fsObject, err := mod.reflectGet(nil, "fs")
	require.NoError(t, err)
	constants, err := mod.reflectGet(fsObject, "constants")
	require.NoError(t, err)

	require.NoError(t, mod.storeValue(sp+8, constants))
	copy(instance.mem[nameOffset:], name)
	require.NoError(t, instance.SetInt64(sp+16, nameOffset))
	require.NoError(t, instance.SetInt64(sp+24, int64(len(name))))

	mod.ValueGet(sp)

	result, err := mod.loadValue(sp + 32)
	require.NoError(t, err)
	assert.IsType(t, float64(0), result)
}

func TestCall(t *testing.T) {
	mod := New(newTestInstance())
	global := mod.values[5].(*jsObject)

	global.properties["echo"] = newjsFunction(func(args []any) any { return args[0] })
	global.properties["notAFunction"] = "value"

	t.Run("calls a registered function", func(t *testing.T) {
		got, err := mod.Call("echo", "payload")
		require.NoError(t, err)
		assert.Equal(t, "payload", got)
	})

	t.Run("unknown function", func(t *testing.T) {
		_, err := mod.Call("unknown")
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("not a function", func(t *testing.T) {
		_, err := mod.Call("notAFunction")
		assert.ErrorContains(t, err, "not a function")
	})
}

type recordingInstance struct {
	*testInstance
	written map[int][]byte
}

func (r *recordingInstance) Write(fd int, data []byte) (int, error) {
	r.written[fd] = append(r.written[fd], data...)
	return len(data), nil
}

func TestFsWriteUsesInstanceWriter(t *testing.T) {
	instance := &recordingInstance{testInstance: newTestInstance(), written: map[int][]byte{}}
	mod := New(instance)

	fsObject, err := mod.reflectGet(nil, "fs")
	require.NoError(t, err)
	write, err := mod.reflectGet(fsObject, "write")
	require.NoError(t, err)

	var callbackArgs []any
	callback := newjsFunction(func(args []any) any {
		callbackArgs = args
		return nil
	})

	_, err = mod.reflectConstruct(write, []any{
		float64(1), &jsUint8Array{data: []byte("hello")}, float64(0), float64(5), nil, callback,
	})
	require.NoError(t, err)

	assert.Equal(t, []byte("hello"), instance.written[1])
	assert.Equal(t, []any{nil, 5}, callbackArgs)
}

func TestFsWriteWithoutWriterReportsError(t *testing.T) {
	mod := New(newTestInstance())

	fsObject, err := mod.reflectGet(nil, "fs")
	require.NoError(t, err)
	write, err := mod.reflectGet(fsObject, "write")
	require.NoError(t, err)

	var callbackArgs []any
	callback := newjsFunction(func(args []any) any {
		callbackArgs = args
		return nil
	})

	_, err = mod.reflectConstruct(write, []any{
		float64(1), &jsUint8Array{data: []byte("hello")}, float64(0), float64(5), nil, callback,
	})
	require.NoError(t, err)

	require.Len(t, callbackArgs, 1)
	assert.Equal(t, jsProperties{"code": "ENOSYS"}, callbackArgs[0])
}

func TestUnimplementedFsFunctionsReturnENOSYS(t *testing.T) {
	mod := New(newTestInstance())

	fsObject, err := mod.reflectGet(nil, "fs")
	require.NoError(t, err)
	open, err := mod.reflectGet(fsObject, "open")
	require.NoError(t, err)

	var callbackArgs []any
	callback := newjsFunction(func(args []any) any {
		callbackArgs = args
		return nil
	})

	_, err = mod.reflectConstruct(open, []any{"/etc/passwd", float64(0), float64(0), callback})
	require.NoError(t, err)

	require.Len(t, callbackArgs, 1)
	assert.Equal(t, jsProperties{"code": "ENOSYS"}, callbackArgs[0])
}
