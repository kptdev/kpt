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

package runtime

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kptdev/kpt/pkg/fn/runtime/wasmexec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

// memoryOnlyModule is the binary encoding of (module (memory (export "mem") 1)).
var memoryOnlyModule = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, // magic and version
	0x05, 0x03, 0x01, 0x00, 0x01, // memory section: one memory with 1 page
	0x07, 0x07, 0x01, 0x03, 'm', 'e', 'm', 0x02, 0x00, // export section: "mem"
}

const testResourceList = `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items: []
`

func writeTempWasm(t *testing.T, content []byte) *FsLoader {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fn.wasm")
	require.NoError(t, os.WriteFile(path, content, 0o600))
	return &FsLoader{Filename: path}
}

// buildGuest compiles testdata/wasmfn for js/wasm with the Go toolchain that is
// running the tests, so that the host is tested against the Go version that
// is actually in use.
func buildGuest(t *testing.T) *FsLoader {
	t.Helper()
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not found in PATH")
	}

	output := filepath.Join(t.TempDir(), "fn.wasm")
	cmd := exec.Command(goBin, "build", "-o", output, ".")
	cmd.Dir = filepath.Join("testdata", "wasmfn")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	return &FsLoader{Filename: output}
}

func TestWazeroFnRunsGoWasmFunction(t *testing.T) {
	loader := buildGuest(t)

	t.Run("returns the resource list produced by the guest", func(t *testing.T) {
		fn, err := NewWazeroFn(loader)
		require.NoError(t, err)

		var out bytes.Buffer
		require.NoError(t, fn.Run(strings.NewReader(testResourceList), &out))

		assert.Equal(t, testResourceList, out.String())
	})

	t.Run("reports the guest errors when the output is not a resource list", func(t *testing.T) {
		fn, err := NewWazeroFn(loader)
		require.NoError(t, err)

		var out bytes.Buffer
		err = fn.Run(strings.NewReader("return-invalid\n"), &out)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid resource list output")
		assert.Contains(t, err.Error(), "guest error details")
		assert.Empty(t, out.String())
	})

	t.Run("is selected by NewWasmFn by default", func(t *testing.T) {
		t.Setenv(WasmRuntimeEnv, "")

		fn, err := NewWasmFn(loader)
		require.NoError(t, err)

		var out bytes.Buffer
		require.NoError(t, fn.Run(strings.NewReader(testResourceList), &out))
		assert.Equal(t, testResourceList, out.String())
	})
}

// TestWazeroFnPrintInsideCallback documents a known limitation: when the guest
// writes to stdout or stderr (for example with fmt.Println or log.Print) while
// it is running inside processResourceList, the host fails with
// "Resume: invalid table access" and returns no usable result. Writing outside
// of the callback, for example at startup, works.
//
// The likely cause is that the write is an fs.write call that makes the host
// invoke a guest callback, which resumes the guest while it is still inside the
// processResourceList call (a nested resume). Remove the skip once that is
// supported.
func TestWazeroFnPrintInsideCallback(t *testing.T) {
	t.Skip("known limitation: printing inside processResourceList fails with a nested resume error")

	fn, err := NewWazeroFn(buildGuest(t))
	require.NoError(t, err)

	input := "print-in-callback\n" + testResourceList
	var out bytes.Buffer
	require.NoError(t, fn.Run(strings.NewReader(input), &out))

	assert.Equal(t, input, out.String())
}

func TestNewWazeroFnInvalidWasm(t *testing.T) {
	_, err := NewWazeroFn(writeTempWasm(t, []byte("this is not wasm")))
	assert.Error(t, err)
}

func TestNewWazeroFnMissingFile(t *testing.T) {
	_, err := NewWazeroFn(&FsLoader{Filename: filepath.Join(t.TempDir(), "missing.wasm")})
	assert.Error(t, err)
}

func TestWazeroFnRunMissingExports(t *testing.T) {
	fn, err := NewWazeroFn(writeTempWasm(t, memoryOnlyModule))
	require.NoError(t, err)

	err = fn.Run(strings.NewReader(testResourceList), &bytes.Buffer{})

	assert.ErrorContains(t, err, "getsp: missing export")
}

func TestNewWasmFnUnsupportedRuntime(t *testing.T) {
	t.Setenv(WasmRuntimeEnv, "bogus")

	_, err := NewWasmFn(&FsLoader{Filename: "unused"})

	assert.ErrorContains(t, err, "unsupported wasm runtime")
}

func TestNewWasmFnWasmtimeIsNoLongerSupported(t *testing.T) {
	t.Setenv(WasmRuntimeEnv, string(Wasmtime))

	_, err := NewWasmFn(&FsLoader{Filename: "unused"})

	assert.ErrorContains(t, err, "no longer supported")
	assert.ErrorContains(t, err, string(Wazero))
}

func TestWazeroFnWrite(t *testing.T) {
	fn := &WazeroFn{}

	_, err := fn.Write(3, []byte("data"))

	assert.ErrorContains(t, err, "invalid file descriptor")
}

func TestWazeroMemory(t *testing.T) {
	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	t.Cleanup(func() { _ = rt.Close(ctx) })

	module, err := rt.Instantiate(ctx, memoryOnlyModule)
	require.NoError(t, err)

	var mem wasmexec.Memory = &wazeroMemory{mem: module.Memory()}
	const memorySize = 64 * 1024

	t.Run("round trips values", func(t *testing.T) {
		require.NoError(t, mem.SetUInt8(10, 0xab))
		require.NoError(t, mem.SetUInt32(16, 0xdeadbeef))
		require.NoError(t, mem.SetInt64(24, -42))
		require.NoError(t, mem.SetFloat64(32, 3.5))

		u32, err := mem.GetUInt32(16)
		require.NoError(t, err)
		assert.Equal(t, uint32(0xdeadbeef), u32)

		i64, err := mem.GetInt64(24)
		require.NoError(t, err)
		assert.Equal(t, int64(-42), i64)

		f64, err := mem.GetFloat64(32)
		require.NoError(t, err)
		assert.Equal(t, 3.5, f64)

		data, err := mem.Range(10, 1)
		require.NoError(t, err)
		assert.Equal(t, []byte{0xab}, data)
	})

	t.Run("range is a view on the module memory", func(t *testing.T) {
		data, err := mem.Range(64, 4)
		require.NoError(t, err)
		copy(data, "wasm")

		got, ok := module.Memory().Read(64, 4)
		require.True(t, ok)
		assert.Equal(t, []byte("wasm"), got)
	})

	t.Run("out of range access reports a fault", func(t *testing.T) {
		_, err := mem.Range(memorySize-1, 2)
		assert.ErrorIs(t, err, wasmexec.ErrFault)
		_, err = mem.GetUInt32(memorySize - 2)
		assert.ErrorIs(t, err, wasmexec.ErrFault)
		_, err = mem.GetInt64(memorySize - 4)
		assert.ErrorIs(t, err, wasmexec.ErrFault)
		_, err = mem.GetFloat64(memorySize - 4)
		assert.ErrorIs(t, err, wasmexec.ErrFault)

		assert.ErrorIs(t, mem.SetUInt8(memorySize, 1), wasmexec.ErrFault)
		assert.ErrorIs(t, mem.SetUInt32(memorySize-2, 1), wasmexec.ErrFault)
		assert.ErrorIs(t, mem.SetInt64(memorySize-4, 1), wasmexec.ErrFault)
		assert.ErrorIs(t, mem.SetFloat64(memorySize-4, 1), wasmexec.ErrFault)
	})
}
