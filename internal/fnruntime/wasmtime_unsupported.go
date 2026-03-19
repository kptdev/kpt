//go:build !cgo || windows

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

package fnruntime

// Stub functions for running without wasmtime support compiled in.
// wasmtime requires cgo, which is not always a viable option.

import (
	"errors"
	"io"
)

const (
	msg = "wasmtime support is not compiled into this binary. Binaries with wasmtime are available at github.com/kptdev/kpt"
)

type WasmtimeFn struct {
}

func NewWasmtimeFn(_ WasmLoader) (*WasmtimeFn, error) {
	return nil, errors.New(msg)
}

func (f *WasmtimeFn) Run(_ io.Reader, _ io.Writer) error {
	return errors.New(msg)
}
