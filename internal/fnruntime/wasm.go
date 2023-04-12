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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/pkg/wasm"
)

type WasmRuntime string

const (
	Wasmtime WasmRuntime = "wasmtime"
	Nodejs   WasmRuntime = "nodejs"

	WasmRuntimeEnv = "KPT_FN_WASM_RUNTIME"

	jsEntrypointFunction = "processResourceList"
)

type WasmFn struct {
	runtimeType WasmRuntime

	wasmtime *WasmtimeFn
	nodejs   *WasmNodejsFn
}

func NewWasmFn(loader WasmLoader) (*WasmFn, error) {
	switch os.Getenv(WasmRuntimeEnv) {
	case string(Nodejs):
		nf, err := NewNodejsFn(loader)
		if err != nil {
			return nil, err
		}
		return &WasmFn{
			runtimeType: Nodejs,
			nodejs:      nf,
		}, nil
	case "":
		fallthrough
	case string(Wasmtime):
		wf, err := NewWasmtimeFn(loader)
		if err != nil {
			return nil, err
		}
		return &WasmFn{
			runtimeType: Wasmtime,
			wasmtime:    wf,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported wasm runtime: %v", os.Getenv(WasmRuntimeEnv))
	}
}

func (f *WasmFn) Run(r io.Reader, w io.Writer) error {
	switch f.runtimeType {
	case Nodejs:
		return f.nodejs.Run(r, w)
	case Wasmtime:
		return f.wasmtime.Run(r, w)
	default:
		return fmt.Errorf("unknown wasm runtime type: %q", f.runtimeType)
	}
}

type WasmLoader interface {
	// getReadCloser returns an io.ReadCloser to read the wasm contents.
	getReadCloser() (io.ReadCloser, error)
	// getFilePath returns a path to the wasm file.
	getFilePath() (string, error)
	// cleanup remove temporary directory or files.
	cleanup() error
}

type OciLoader struct {
	cacheDir string
	image    string
	tempDir  string
}

var _ WasmLoader = &OciLoader{}

func NewOciLoader(cacheDir, image string) *OciLoader {
	return &OciLoader{
		cacheDir: cacheDir,
		image:    image,
	}
}

func (o *OciLoader) getReadCloser() (io.ReadCloser, error) {
	storage, err := wasm.NewClient(o.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("unable to create a storage client: %w", err)
	}
	rc, err := storage.LoadWasm(context.TODO(), o.image)
	if err != nil {
		return nil, fmt.Errorf("unable to load image from %v: %w", o.image, err)
	}
	return rc, nil
}

func (o *OciLoader) getFilePath() (string, error) {
	rc, err := o.getReadCloser()
	if err != nil {
		return "", fmt.Errorf("unable to get reader from OCI image: %w", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("unable to read wasm content from reader: %w", err)
	}
	o.tempDir, err = os.MkdirTemp(o.cacheDir, "oci-loader-")
	if err != nil {
		return "", fmt.Errorf("unable to create temp dir in %v: %w", o.cacheDir, err)
	}
	wasmFile := filepath.Join(o.tempDir, "fn.wasm")
	err = os.WriteFile(wasmFile, data, 0644)
	if err != nil {
		return "", fmt.Errorf("unable to write wasm content to %v: %w", wasmFile, err)
	}
	return wasmFile, nil
}

func (o *OciLoader) cleanup() error {
	if o.tempDir != "" {
		return os.RemoveAll(o.tempDir)
	}
	return nil
}

type FsLoader struct {
	Filename string
}

var _ WasmLoader = &FsLoader{}

func (f *FsLoader) getFilePath() (string, error) {
	return f.Filename, nil
}

func (f *FsLoader) getReadCloser() (io.ReadCloser, error) {
	fi, err := os.Open(f.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %v: %w", f.Filename, err)
	}
	return fi, nil
}

func (f *FsLoader) cleanup() error {
	return nil
}
