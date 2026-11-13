// This file is derived from github.com/prep/wasmexec
// (https://github.com/prep/wasmexec, commit 6554945c1dec), which is
// Copyright (c) 2022 Maurice Nonnekes and licensed under the MIT License
// (see the LICENSE file in this directory).
//
// Modified by The kpt Authors in 2026:
// - added the jsUndefined type, so that missing properties can be told apart
//   from null.
// - errorResponse and errorCallback no longer take an errno parameter, because
//   they were only ever used with ENOSYS.

package wasmexec

// errno describes an error "number".
type errno string

// This errno list is a subset of the errors in syscall/tables_js.go.
const (
	eNOSYS errno = "ENOSYS"
)

// errorResponse returns an ENOSYS callback response.
func errorResponse() []any {
	return []any{jsProperties{"code": string(eNOSYS)}}
}

// errorCallback returns a function that calls the callback with an ENOSYS
// errno code.
func errorCallback() *jsFunction {
	return &jsFunction{
		fn: func(args []any) any {
			if len(args) == 0 {
				return nil
			}

			// The last item in the list should be a callback function, according to
			// the fsCall() function in syscall/fs.js.go.
			callback, ok := args[len(args)-1].(*jsFunction)
			if !ok {
				return nil
			}

			callback.fn(errorResponse())
			return nil
		},
	}
}

// jsFunction describes the constructor of an jsObject.
type jsFunction struct {
	name string
	fn   func(args []any) any
}

// newjsFunction returns a new function.
func newjsFunction(fn func(args []any) any) *jsFunction {
	return &jsFunction{fn: fn}
}

// Name returns the name of the constructor type.
func (fn jsFunction) Name() string {
	return fn.name
}

// jsProperties describe the properties on an object. This can either be a
// function or a value.
type jsProperties map[string]any

// jsObject describes a JSON object.
type jsObject struct {
	properties jsProperties
}

// jsArray describes an array of elements.
type jsArray struct {
	elements []any
}

// jsUint8Array describes a byte slice.
type jsUint8Array struct {
	data []byte
}

// jsString represents a stored string.
type jsString struct {
	data string
}

// jsUndefined represents the JavaScript value "undefined", which is distinct
// from null (nil). Go's syscall package relies on that distinction.
type jsUndefined struct{}

var undefined = jsUndefined{}
