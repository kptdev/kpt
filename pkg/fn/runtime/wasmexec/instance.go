// This file is derived from github.com/prep/wasmexec
// (https://github.com/prep/wasmexec, commit 6554945c1dec), which is
// Copyright (c) 2022 Maurice Nonnekes and licensed under the MIT License
// (see the LICENSE file in this directory).
//
// This file is copied without modifications.

package wasmexec

// Instance describes an instance of a Wasm module.
type Instance interface {
	Memory

	// Get the SP value.
	GetSP() (uint32, error)

	// Resume the execution of Go code until it needs to wait for an event.
	Resume() error
}
