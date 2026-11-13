// Copyright 2022,2026 The kpt Authors
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
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/kptdev/kpt/pkg/fn/runtime/wasmexec"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// WazeroFn runs a Go js/wasm function using the pure Go wazero runtime. The
// "go" imports (the Go wasm_exec.js shim) are provided by wasmexec.
//
// Known limitation: a function that writes to stdout or stderr (fmt.Println,
// log.Print, ...) while it is running inside processResourceList makes the run
// fail with a nested resume error. Output written outside of that call, for
// example at startup, works. See TestWazeroFnPrintInsideCallback.
type WazeroFn struct {
	wasmexec.Memory

	ctx      context.Context
	instance api.Module
	gomod    *wasmexec.Module

	spFn     api.Function
	resumeFn api.Function

	loader WasmLoader
}

func NewWazeroFn(loader WasmLoader) (*WazeroFn, error) {
	f := &WazeroFn{
		ctx:    context.Background(),
		loader: loader,
	}
	wasmFileReadCloser, err := loader.getReadCloser()
	if err != nil {
		return nil, err
	}
	defer wasmFileReadCloser.Close()
	data, err := io.ReadAll(wasmFileReadCloser)
	if err != nil {
		return nil, fmt.Errorf("unable to read wasm content from reader: %w", err)
	}

	rt := wazero.NewRuntime(f.ctx)

	f.gomod = wasmexec.New(f)
	if err = importGoFuncs(f.ctx, rt, f.gomod); err != nil {
		return nil, err
	}

	// The Go runtime is started explicitly through the "run" export.
	config := wazero.NewModuleConfig().WithStartFunctions()
	if f.instance, err = rt.InstantiateWithConfig(f.ctx, data, config); err != nil {
		return nil, err
	}

	return f, nil
}

// importGoFuncs defines the host modules that Go js/wasm binaries import.
func importGoFuncs(ctx context.Context, rt wazero.Runtime, mod *wasmexec.Module) error {
	funcs := map[string]func(uint32){
		"runtime.wasmExit":              mod.WasmExit,
		"runtime.wasmWrite":             mod.WasmWrite,
		"runtime.resetMemoryDataView":   mod.ResetMemoryDataView,
		"runtime.nanotime1":             mod.Nanotime1,
		"runtime.walltime":              mod.Walltime,
		"runtime.scheduleTimeoutEvent":  mod.ScheduleTimeoutEvent,
		"runtime.clearTimeoutEvent":     mod.ClearTimeoutEvent,
		"runtime.getRandomData":         mod.GetRandomData,
		"syscall/js.finalizeRef":        mod.FinalizeRef,
		"syscall/js.stringVal":          mod.StringVal,
		"syscall/js.valueGet":           mod.ValueGet,
		"syscall/js.valueSet":           mod.ValueSet,
		"syscall/js.valueDelete":        mod.ValueDelete,
		"syscall/js.valueIndex":         mod.ValueIndex,
		"syscall/js.valueSetIndex":      mod.ValueSetIndex,
		"syscall/js.valueCall":          mod.ValueCall,
		"syscall/js.valueInvoke":        mod.ValueInvoke,
		"syscall/js.valueNew":           mod.ValueNew,
		"syscall/js.valueLength":        mod.ValueLength,
		"syscall/js.valuePrepareString": mod.ValuePrepareString,
		"syscall/js.valueLoadString":    mod.ValueLoadString,
		"syscall/js.valueInstanceOf":    mod.ValueInstanceOf,
		"syscall/js.copyBytesToGo":      mod.CopyBytesToGo,
		"syscall/js.copyBytesToJS":      mod.CopyBytesToJS,
		"debug":                         mod.Debug,
	}

	// Go before 1.21 imports these functions from "go", later versions from "gojs".
	for _, moduleName := range []string{"go", "gojs"} {
		builder := rt.NewHostModuleBuilder(moduleName)
		for name, fn := range funcs {
			builder = builder.NewFunctionBuilder().WithFunc(fn).Export(name)
		}
		if _, err := builder.Instantiate(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Run runs the executable file which reads the input from r and
// writes the output to w.
func (f *WazeroFn) Run(r io.Reader, w io.Writer) error {
	// Make the memory export accessible by the imports.
	mem := f.instance.Memory()
	if mem == nil {
		return errors.New("unable to find memory export")
	}
	f.Memory = &wazeroMemory{mem: mem}

	if f.spFn = f.instance.ExportedFunction("getsp"); f.spFn == nil {
		return errors.New("getsp: missing export")
	}
	if f.resumeFn = f.instance.ExportedFunction("resume"); f.resumeFn == nil {
		return errors.New("resume: missing export")
	}
	runFn := f.instance.ExportedFunction("run")
	if runFn == nil {
		return errors.New("run: missing export")
	}

	// Set the args and the environment variables.
	argc, argv, err := wasmexec.SetArgs(f.Memory, []string{"krm-fn-wasm-wazero"}, []string{})
	if err != nil {
		return err
	}

	// Call the "run" function. This starts the program.
	if _, err = runFn.Call(f.ctx, api.EncodeI32(argc), api.EncodeI32(argv)); err != nil {
		return err
	}

	resourceList, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	result, err := f.gomod.Call(jsEntrypointFunction, string(resourceList))
	if err != nil {
		return fmt.Errorf("unable to invoke %v: %v", jsEntrypointFunction, err)
	}

	// We expect `result` to be a *wasmexec.jsString (which is not exportable) with
	// the following definition: type jsString struct { data string }. It will look
	// like `&{realPayload}`
	resultStr := fmt.Sprintf("%s", result)
	resultStr = resultStr[2 : len(resultStr)-1]
	// Try to parse the output as yaml.
	resourceListOutput, err := yaml.Parse(resultStr)
	if err != nil {
		additionalErrorMessage, errorResultRetrievalErr := retrieveError(f, resourceList)
		if errorResultRetrievalErr != nil {
			return errorResultRetrievalErr
		}
		return fmt.Errorf("parsing output resource list with content: %q\n%w\n%s", resultStr, err, additionalErrorMessage)
	}
	if resourceListOutput.GetKind() != "ResourceList" {
		additionalErrorMessage, errorResultRetrievalErr := retrieveError(f, resourceList)
		if errorResultRetrievalErr != nil {
			return errorResultRetrievalErr
		}
		return fmt.Errorf("invalid resource list output from wasm library; got %q\n%s", resultStr, additionalErrorMessage)
	}
	if _, err = w.Write([]byte(resultStr)); err != nil {
		return fmt.Errorf("unable to write the output resource list: %w", err)
	}
	return f.loader.cleanup()
}

func retrieveError(f *WazeroFn, resourceList []byte) (string, error) {
	errResult, err := f.gomod.Call(jsEntrypointFunction+"Errors", string(resourceList))
	if err != nil {
		return "", fmt.Errorf("unable to retrieve additional error message from function: %w", err)
	}
	return fmt.Sprintf("%s", errResult), nil
}

var _ wasmexec.Instance = &WazeroFn{}

func (f *WazeroFn) GetSP() (uint32, error) {
	results, err := f.spFn.Call(f.ctx)
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, errors.New("getsp: no sp value returned")
	}

	sp := api.DecodeI32(results[0])
	if sp >= 0 {
		return uint32(sp), nil
	}

	return 0, fmt.Errorf("getsp: %T: expected a positive return value", sp)
}

func (f *WazeroFn) Resume() error {
	_, err := f.resumeFn.Call(f.ctx)
	return err
}

// Write implements the wasmexec.fdWriter interface.
func (f *WazeroFn) Write(fd int, b []byte) (n int, err error) {
	switch fd {
	case 1, 2:
		n, err = os.Stdout.Write(b)
	default:
		err = fmt.Errorf("%d: invalid file descriptor", fd)
	}

	return n, err
}

// Error implements the wasmexec.errorLogger interface
func (f *WazeroFn) Error(format string, params ...any) {
	log.Printf("ERROR: "+format+"\n", params...)
}

// wazeroMemory adapts a wazero api.Memory to the wasmexec.Memory interface.
type wazeroMemory struct {
	mem api.Memory
}

var _ wasmexec.Memory = &wazeroMemory{}

func (m *wazeroMemory) Range(offset, length uint32) ([]byte, error) {
	data, ok := m.mem.Read(offset, length)
	if !ok {
		return nil, wasmexec.ErrFault
	}
	return data, nil
}

func (m *wazeroMemory) GetUInt32(offset uint32) (uint32, error) {
	val, ok := m.mem.ReadUint32Le(offset)
	if !ok {
		return 0, wasmexec.ErrFault
	}
	return val, nil
}

func (m *wazeroMemory) GetInt64(offset uint32) (int64, error) {
	val, ok := m.mem.ReadUint64Le(offset)
	if !ok {
		return 0, wasmexec.ErrFault
	}
	return int64(val), nil
}

func (m *wazeroMemory) GetFloat64(offset uint32) (float64, error) {
	val, ok := m.mem.ReadFloat64Le(offset)
	if !ok {
		return 0, wasmexec.ErrFault
	}
	return val, nil
}

func (m *wazeroMemory) SetUInt8(offset uint32, val uint8) error {
	if !m.mem.WriteByte(offset, val) {
		return wasmexec.ErrFault
	}
	return nil
}

func (m *wazeroMemory) SetUInt32(offset, val uint32) error {
	if !m.mem.WriteUint32Le(offset, val) {
		return wasmexec.ErrFault
	}
	return nil
}

func (m *wazeroMemory) SetInt64(offset uint32, val int64) error {
	if !m.mem.WriteUint64Le(offset, uint64(val)) {
		return wasmexec.ErrFault
	}
	return nil
}

func (m *wazeroMemory) SetFloat64(offset uint32, val float64) error {
	if !m.mem.WriteFloat64Le(offset, val) {
		return wasmexec.ErrFault
	}
	return nil
}
