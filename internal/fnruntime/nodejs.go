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
	"fmt"
	"io"
	"os"
	"path/filepath"

	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
)

const (
	WasmPathEnv = "KPT_FN_WASM_PATH"
)

type WasmNodejsFn struct {
	NodeJsRunner *ExecFn
	loader       WasmLoader
}

func NewNodejsFn(loader WasmLoader) (*WasmNodejsFn, error) {
	cacheDir := filepath.Join(os.TempDir(), "kpt-wasm-fn")
	err := os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create cache dir: %w", err)
	}
	tempDir, err := os.MkdirTemp(cacheDir, "nodejs-")
	if err != nil {
		return nil, fmt.Errorf("unable to create temp dir: %w", err)
	}
	jsPath := filepath.Join(tempDir, "kpt-fn-wasm-glue-runner.js")
	if err = os.WriteFile(jsPath, []byte(golangWasmJSCode+glueCode), 0644); err != nil {
		return nil, fmt.Errorf("unable to write the js glue code file: %w", err)
	}

	wasmFile, err := loader.getFilePath()
	if err != nil {
		return nil, err
	}

	f := &WasmNodejsFn{
		NodeJsRunner: &ExecFn{
			Path: "node",
			Args: []string{jsPath},
			Env: map[string]string{
				WasmPathEnv: wasmFile,
			},
			FnResult: &fnresult.Result{},
		},
		loader: loader,
	}
	return f, nil
}

func (f *WasmNodejsFn) Run(r io.Reader, w io.Writer) error {
	if err := f.NodeJsRunner.Run(r, w); err != nil {
		return fmt.Errorf("failed to run wasm with node.js: %w", err)
	}
	return f.loader.cleanup()
}
