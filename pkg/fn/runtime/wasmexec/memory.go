// This file is derived from github.com/prep/wasmexec
// (https://github.com/prep/wasmexec, commit 6554945c1dec), which is
// Copyright (c) 2022 Maurice Nonnekes and licensed under the MIT License
// (see the LICENSE file in this directory).
//
// Modified by The kpt Authors in 2026:
// - removed the byte slice based Memory implementation (NewMemory), because
//   kpt provides a Memory implementation on top of the wazero runtime.

package wasmexec

import (
	"errors"
)

// ErrFault is returned whenever memory was accessed that was not addressable.
var ErrFault = errors.New("bad address")

// Memory describes an instantiated module's memory.
type Memory interface {
	Range(offset, length uint32) ([]byte, error)

	GetUInt32(offset uint32) (uint32, error)
	GetInt64(offset uint32) (int64, error)
	GetFloat64(offset uint32) (float64, error)
	SetUInt8(offset uint32, val uint8) error
	SetUInt32(offset, val uint32) error
	SetInt64(offset uint32, val int64) error
	SetFloat64(offset uint32, val float64) error
}

// wasmMinDataAddr
const wasmMinDataAddr = 4096 + 8192

// SetArgs sets the specified arguments and environment variables.
func SetArgs(mem Memory, args, envs []string) (int32, int32, error) {
	offset := uint32(4096)

	strPtr := func(value string) (uint32, error) {
		ptr := offset
		bytes := []byte(value + "\000")
		length := uint32(len(bytes))

		data, err := mem.Range(ptr, length)
		if err != nil {
			return 0, err
		}

		_ = copy(data, bytes)

		offset += length
		if offset%8 != 0 {
			offset += 8 - (offset % 8)
		}

		return ptr, nil
	}

	// Process the command line arguments.
	var argvPtrs []uint32
	for _, arg := range args {
		ptr, err := strPtr(arg)
		if err != nil {
			return 0, 0, err
		}

		argvPtrs = append(argvPtrs, ptr)
	}
	argvPtrs = append(argvPtrs, 0)

	// Process the environment variables.
	for _, env := range envs {
		ptr, err := strPtr(env)
		if err != nil {
			return 0, 0, err
		}

		argvPtrs = append(argvPtrs, ptr)
	}
	argvPtrs = append(argvPtrs, 0)

	// Write the list of pointers. The start of this list is what argv points to.
	argv := offset
	for _, ptr := range argvPtrs {
		if err := mem.SetUInt32(offset, ptr); err != nil {
			return 0, 0, err
		}
		if err := mem.SetUInt32(offset+4, 0); err != nil {
			return 0, 0, err
		}

		offset += 8
	}

	// Make sure the args + environment variables have not overwritten the
	// data section.
	if offset >= wasmMinDataAddr {
		return 0, 0, errors.New("total length of command line and environment variables exceeds limit")
	}

	return int32(len(args)), int32(argv), nil
}
