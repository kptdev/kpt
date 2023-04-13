//go:build !cgo

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

// Stub functions for running without wasmtime support compiled in.
// wasmtime requires cgo, which is not always a viable option.

import (
	"fmt"
	"io"
)

const (
	msg = "wasmtime support is not complied into this binary. Binaries with wasmtime is avilable at github.com/GoogleContainerTools/kpt"
)

type WasmtimeFn struct {
}

func NewWasmtimeFn(loader WasmLoader) (*WasmtimeFn, error) {
	return nil, fmt.Errorf(msg)
}

func (f *WasmtimeFn) Run(r io.Reader, w io.Writer) error {
	return fmt.Errorf(msg)
}
