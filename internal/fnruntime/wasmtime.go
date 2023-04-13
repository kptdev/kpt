//go:build cgo

// Copyright 2022 The kpt Authors
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

package fnruntime

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	wasmtime "github.com/bytecodealliance/wasmtime-go"
	"github.com/prep/wasmexec"
	"github.com/prep/wasmexec/wasmtimexec"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type WasmtimeFn struct {
	wasmexec.Memory
	*wasmtime.Instance
	store *wasmtime.Store

	gomod *wasmexec.Module

	spFn     *wasmtime.Func
	resumeFn *wasmtime.Func

	loader WasmLoader
}

func NewWasmtimeFn(loader WasmLoader) (*WasmtimeFn, error) {
	f := &WasmtimeFn{
		loader: loader,
	}
	wasmFileReadCloser, err := loader.getReadCloser()
	if err != nil {
		return nil, err
	}
	defer wasmFileReadCloser.Close()
	// Create a module out of the Wasm file.
	data, err := io.ReadAll(wasmFileReadCloser)
	if err != nil {
		return nil, fmt.Errorf("unable to read wasm content from reader: %w", err)
	}

	// Create the engine and store.
	config := wasmtime.NewConfig()
	err = config.CacheConfigLoadDefault()
	if err != nil {
		return nil, fmt.Errorf("failed to config cache in wasmtime")
	}
	engine := wasmtime.NewEngineWithConfig(config)

	module, err := wasmtime.NewModule(engine, data)
	if err != nil {
		return nil, err
	}
	f.store = wasmtime.NewStore(engine)

	linker := wasmtime.NewLinker(engine)
	f.gomod, err = wasmtimexec.Import(f.store, linker, f)
	if err != nil {
		return nil, err
	}

	// Create an instance of the module.
	if f.Instance, err = linker.Instantiate(f.store, module); err != nil {
		return nil, err
	}

	return f, nil
}

// Run runs the executable file which reads the input from r and
// writes the output to w.
func (f *WasmtimeFn) Run(r io.Reader, w io.Writer) error {
	// Fetch the memory export and set it on the instance, making the memory
	// accessible by the imports.
	ext := f.GetExport(f.store, "mem")
	if ext == nil {
		return errors.New("unable to find memory export")
	}

	mem := ext.Memory()
	if mem == nil {
		return errors.New("mem: export is not memory")
	}

	f.Memory = wasmexec.NewMemory(mem.UnsafeData(f.store))

	// Fetch the getsp function and reference it on the instance.
	spFn := f.GetExport(f.store, "getsp")
	if spFn == nil {
		return errors.New("getsp: missing export")
	}

	if f.spFn = spFn.Func(); f.spFn == nil {
		return errors.New("getsp: export is not a function")
	}

	// Fetch the resume function and reference it on the instance.
	resumeFn := f.GetExport(f.store, "resume")
	if resumeFn == nil {
		return errors.New("resume: missing export")
	}

	if f.resumeFn = resumeFn.Func(); f.resumeFn == nil {
		return errors.New("resume: export is not a function")
	}

	// Set the args and the environment variables.
	argc, argv, err := wasmexec.SetArgs(f.Memory, []string{"kpt-fn-wasm-wasmtime"}, []string{})
	if err != nil {
		return err
	}

	// Fetch the "run" function and call it. This starts the program.
	runFn := f.GetFunc(f.store, "run")
	if runFn == nil {
		return errors.New("run: missing export")
	}

	if _, err = runFn.Call(f.store, argc, argv); err != nil {
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

func retrieveError(f *WasmtimeFn, resourceList []byte) (string, error) {
	errResult, err := f.gomod.Call(jsEntrypointFunction+"Errors", string(resourceList))
	if err != nil {
		return "", fmt.Errorf("unable to retrieve additional error message from function: %w", err)
	}
	return fmt.Sprintf("%s", errResult), nil
}

var _ wasmexec.Instance = &WasmtimeFn{}

func (f *WasmtimeFn) GetSP() (uint32, error) {
	val, err := f.spFn.Call(f.store)
	if err != nil {
		return 0, err
	}

	sp, ok := val.(int32)
	if !ok {
		return 0, fmt.Errorf("getsp: %T: expected an int32 return value", sp)
	}

	return uint32(sp), nil
}

func (f *WasmtimeFn) Resume() error {
	_, err := f.resumeFn.Call(f.store)
	return err
}

// Write implements the wasmexec.fdWriter interface.
func (f *WasmtimeFn) Write(fd int, b []byte) (n int, err error) {
	switch fd {
	case 1, 2:
		n, err = os.Stdout.Write(b)
	default:
		err = fmt.Errorf("%d: invalid file descriptor", fd)
	}

	return n, err
}

// Error implements the wasmexec.errorLogger interface
func (f *WasmtimeFn) Error(format string, params ...interface{}) {
	log.Printf("ERROR: "+format+"\n", params...)
}
