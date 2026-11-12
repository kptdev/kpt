// This file is derived from github.com/prep/wasmexec
// (https://github.com/prep/wasmexec, commit 6554945c1dec), which is
// Copyright (c) 2022 Maurice Nonnekes and licensed under the MIT License
// (see the LICENSE file in this directory).
//
// Modified by The kpt Authors in 2026:
// - reflectGet returns an "undefined" value for missing properties instead of
//   nil, and storeValue encodes it as undefined. Go 1.23+'s syscall package
//   requires this to initialize.
// - ResetMemoryDataView is a no-op instead of logging "NOT IMPLEMENTED".

package wasmexec

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"syscall"
	"time"
)

// nanHead is the NaN-header for values that are not a number, but an ID.
const nanHead = 0x7FF80000

// NaN describes a not-a-number value.
var NaN = math.NaN()

// invokeContext keeps track of the response from the guest during an Invoke
// call.
type invokeContext struct {
	guestResp []byte
	guestErr  string
	success   chan bool
}

func newInvokeContext() *invokeContext {
	return &invokeContext{success: make(chan bool, 1)}
}

// debugLogger describes an instance that has implemented a debug logger.
type debugLogger interface {
	Debug(format string, params ...any)
}

// errorLogger describes an instance that has implemented an error logger.
type errorLogger interface {
	Error(format string, params ...any)
}

// fdWriter describes an instance that has implemented os.Write.
type fdWriter interface {
	Write(fd int, data []byte) (n int, err error)
}

// exiter describes an Instance that has implemented an Exit method.
type exiter interface {
	Exit(code int)
}

// hostCaller describes an instance that has implemented the waPC HostCall method.
type hostCaller interface {
	HostCall(string, string, string, []byte) ([]byte, error)
}

// Module implements the JavaScript imports that a Go program compiled with
// GOOS=js expects.
type Module struct {
	instance      Instance
	invokeContext *invokeContext

	debugLog debugLogger
	errorLog errorLogger
	writer   fdWriter
	exit     exiter
	waPC     hostCaller

	idcounter uint32
	ids       map[any]uint32
	values    map[uint32]any
	refcounts map[uint32]int32
}

// New returns a new Module.
func New(instance Instance) *Module {
	debugLog, _ := instance.(debugLogger)
	errorLog, _ := instance.(errorLogger)
	writer, _ := instance.(fdWriter)
	exit, _ := instance.(exiter)
	waPC, _ := instance.(hostCaller)

	var mod *Module
	mod = &Module{
		instance: instance,

		debugLog: debugLog,
		errorLog: errorLog,
		writer:   writer,
		exit:     exit,
		waPC:     waPC,

		idcounter: 10,
		refcounts: make(map[uint32]int32),
		ids: map[any]uint32{
			NaN:        0,
			float64(0): 1,
			nil:        2,
			true:       3,
			false:      4,
		},
		values: map[uint32]any{
			0: NaN,
			1: float64(0),
			2: nil,
			3: true,
			4: false,

			// global.
			5: &jsObject{
				properties: jsProperties{
					"Array": &jsFunction{
						name: "Array",
						fn: func([]any) any {
							return &jsArray{}
						},
					},

					"Date": &jsFunction{
						fn: func([]any) any {
							return &jsObject{
								properties: jsProperties{
									"getTimezoneOffset": &jsFunction{
										fn: func(args []any) any {
											t := time.Now()
											_, offset := t.Zone()
											return (offset / 60) * -1
										},
									},
								},
							}
						},
					},

					"Object": &jsFunction{
						name: "Object",
						fn: func([]any) any {
							return &jsObject{properties: make(jsProperties)}
						},
					},

					"Uint8Array": &jsFunction{
						name: "Uint8Array",
						fn: func(args []any) any {
							if len(args) == 0 {
								return []byte{}
							}

							length, ok := args[0].(float64)
							if !ok {
								return []byte{}
							}

							return &jsUint8Array{
								data: make([]byte, uint32(length)),
							}
						},
					},

					"crypto": &jsObject{
						properties: jsProperties{
							"getRandomValues": &jsFunction{
								fn: func(args []any) any {
									if len(args) != 1 {
										mod.error("crypto.getRandomValues: %d: invalid number of arguments", len(args))
										return 0
									}

									a, ok := args[0].(*jsUint8Array)
									if !ok {
										mod.error("crypto.getRandomValues: %T: not type jsUint8Array", args[0])
										return 0
									}

									n, err := rand.Read(a.data)
									if err != nil {
										mod.error("crypto.getRandomValues: %v", err)
										return 0
									}

									return n
								},
							},
						},
					},

					"fs": &jsObject{
						properties: jsProperties{
							"constants": jsProperties{
								"O_WRONLY": syscall.O_WRONLY,
								"O_RDWR":   syscall.O_RDWR,
								"O_CREAT":  syscall.O_CREAT,
								"O_TRUNC":  syscall.O_TRUNC,
								"O_APPEND": syscall.O_APPEND,
								"O_EXCL":   syscall.O_EXCL,
							},

							"write": &jsFunction{
								fn: func(args []any) any {
									if len(args) != 6 {
										mod.error("fs.write: %d: invalid number of arguments", len(args))
										return nil
									}

									val, ok := args[0].(float64)
									if !ok {
										mod.error("fs.write: %T: not type float64", args[0])
										return nil
									}
									fd := int(val)

									buf, ok := args[1].(*jsUint8Array)
									if !ok {
										mod.error("fs.write: %T: not type jsUint8Array", args[1])
										return nil
									}

									/*
										offset, ok := args[2].(int)
										if !ok {
											mod.error("fs.write: %T: not type int", args[2])
											return 9
										}

										val, ok = args[3].(float64)
										if !ok {
											mod.error("fs.write: %T: not type float64", args[3])
											return 9
										}
										length := int(val)

										var position int64
										if args[4] != nil {
											val, ok = args[4].(float64)
											if !ok {
												mod.error("fs.write: %T: not type float64", args[4])
												return 9
											}

											position = int64(val)
										}
									*/

									callback, ok := args[5].(*jsFunction)
									if !ok {
										mod.error("fs.write: %T: not type jsFunction", args[5])
										return nil
									}

									n, err := mod.write(fd, buf.data)
									if err != nil {
										callback.fn(errorResponse(eNOSYS))
										return nil
									}

									callback.fn([]any{nil, n})
									return nil
								},
							},

							"chmod":     errorCallback(eNOSYS),
							"chown":     errorCallback(eNOSYS),
							"close":     errorCallback(eNOSYS),
							"fchmod":    errorCallback(eNOSYS),
							"fchown":    errorCallback(eNOSYS),
							"fstat":     errorCallback(eNOSYS),
							"fsync":     errorCallback(eNOSYS),
							"ftruncate": errorCallback(eNOSYS),
							"lchown":    errorCallback(eNOSYS),
							"link":      errorCallback(eNOSYS),
							"lstat":     errorCallback(eNOSYS),
							"mkdir":     errorCallback(eNOSYS),
							"open":      errorCallback(eNOSYS),
							"read":      errorCallback(eNOSYS),
							"readdir":   errorCallback(eNOSYS),
							"readlink":  errorCallback(eNOSYS),
							"rename":    errorCallback(eNOSYS),
							"rmdir":     errorCallback(eNOSYS),
							"stat":      errorCallback(eNOSYS),
							"symlink":   errorCallback(eNOSYS),
							"truncate":  errorCallback(eNOSYS),
							"unlink":    errorCallback(eNOSYS),
							"utimes":    errorCallback(eNOSYS),
						},
					},

					"process": &jsObject{
						properties: jsProperties{
							"getuid":    newjsFunction(func([]any) any { return -1 }),
							"getgid":    newjsFunction(func([]any) any { return -1 }),
							"geteuid":   newjsFunction(func([]any) any { return -1 }),
							"getegid":   newjsFunction(func([]any) any { return -1 }),
							"getgroups": newjsFunction(func([]any) any { return []any{} }),
							"pid":       -1,
							"ppid":      -1,

							// Unsure how to "throw" an eNOSYS, so return a reasonable value instead.
							"umask": newjsFunction(func([]any) any { return 0o22 }),
							"cwd":   newjsFunction(func(args []any) any { return "/" }),
							"chdir": newjsFunction(func([]any) any { return nil }),
						},
					},

					// waPC.
					"wapc": &jsObject{
						properties: jsProperties{
							"__guest_response": &jsFunction{
								fn: func(args []any) any {
									if len(args) != 1 {
										return nil
									}

									if resp, ok := args[0].(*jsUint8Array); ok {
										mod.invokeContext.guestResp = resp.data
										mod.invokeContext.success <- true
									}

									return nil
								},
							},
							"__guest_error": &jsFunction{
								fn: func(args []any) any {
									if len(args) != 1 {
										return nil
									}

									if resp, ok := args[0].(*jsUint8Array); ok {
										mod.invokeContext.guestErr = string(resp.data)
										mod.invokeContext.success <- false
									}

									return nil
								},
							},
							"__host_call": &jsFunction{
								fn: func(args []any) any {
									resp, err := func() ([]byte, error) {
										if mod.waPC == nil {
											return nil, errors.New("no waPC host support")
										}

										if len(args) != 4 {
											return nil, fmt.Errorf("%d: unexpected length of arguments for __host_call", len(args))
										}

										binding, ok := args[0].(*jsUint8Array)
										if !ok {
											return nil, fmt.Errorf("%T: unexpected type for binding parameter", args[0])
										}

										namespace, ok := args[1].(*jsUint8Array)
										if !ok {
											return nil, fmt.Errorf("%T: unexpected type for namespace parameter", args[1])
										}

										operation, ok := args[2].(*jsUint8Array)
										if !ok {
											return nil, fmt.Errorf("%T: unexpected type for operation parameter", args[2])
										}

										payload, ok := args[3].(*jsUint8Array)
										if !ok {
											return nil, fmt.Errorf("%T: unexpected type for payload parameter", args[3])
										}

										return mod.waPC.HostCall(string(binding.data), string(namespace.data), string(operation.data), payload.data)
									}()
									if err != nil {
										return []any{nil, &jsString{data: err.Error()}}
									}

									return []any{&jsUint8Array{data: resp}, nil}
								},
							},
						},
					},
				},
			},

			// jsGo.
			6: &jsObject{
				properties: jsProperties{
					"_pendingEvent": nil,

					// This is called by js.FuncOf().
					"_makeFuncWrapper": &jsFunction{
						fn: func(args []any) any {
							if len(args) == 0 {
								return nil
							}

							id := args[0]

							return &jsFunction{
								fn: func(args []any) any {
									event := &jsObject{
										properties: jsProperties{
											"id": id,
											// "this": mod.values[6].(*jsObject),
											"this": nil,
											"args": &jsArray{elements: args},
										},
									}

									mod.values[6].(*jsObject).properties["_pendingEvent"] = event
									if err := mod.instance.Resume(); err != nil {
										mod.error("_makeFuncWrapper: Resume: %v", err)
										return nil
									}

									return event.properties["result"]
								},
							}
						},
					},
				},
			},
		},
	}

	return mod
}

// Call a function created by js.FuncOf().
func (mod *Module) Call(name string, args ...any) (any, error) {
	obj, ok := mod.values[5].(*jsObject)
	if !ok {
		return nil, errors.New("global not an object")
	}

	prop, ok := obj.properties[name]
	if !ok {
		return nil, fmt.Errorf("%s: not found", name)
	}

	fn, ok := prop.(*jsFunction)
	if !ok {
		return nil, fmt.Errorf("%s: not a function", name)
	}

	return fn.fn(args), nil
}

// Invoke calls operation with the specified payload and returns a []byte payload.
func (mod *Module) Invoke(operation string, payload []byte) ([]byte, error) {
	mod.invokeContext = newInvokeContext()

	_, err := mod.Call("__guest_call", operation, payload)
	if err != nil {
		return nil, err
	}

	if ok := <-mod.invokeContext.success; !ok {
		return nil, errors.New(mod.invokeContext.guestErr)
	}

	return mod.invokeContext.guestResp, nil
}

// ****************************************************************************
// **************************** [ Helper methods ] ****************************
// ****************************************************************************

func (mod *Module) debug(format string, params ...any) {
	if mod.debugLog != nil {
		mod.debugLog.Debug(format, params...)
	}
}

func (mod *Module) error(format string, params ...any) {
	if mod.errorLog != nil {
		mod.errorLog.Error(format, params...)
	}
}

func (mod *Module) write(fd int, data []byte) (int, error) {
	if mod.writer == nil {
		return 0, errors.New("no writer available")
	}

	return mod.writer.Write(fd, data)
}

// TODO: Perhaps we need a better scheme of assigning IDs to in-memory objects.
func (mod *Module) getID() uint32 {
	id := mod.idcounter
	mod.idcounter++

	return id
}

// loadValue loads either a number from the specified address, or it loads an
// object ID from the address and fetches that value from the stored values.
func (mod *Module) loadValue(addr uint32) (any, error) {
	f, err := mod.instance.GetFloat64(addr)
	switch {
	case err != nil:
		return nil, err
	case f == 0:
		return nil, nil
	case !math.IsNaN(f):
		return f, nil
	}

	id, err := mod.instance.GetUInt32(addr)
	if err != nil {
		return nil, err
	}

	mod.debug("   loadValue(id=%v)", id)

	return mod.values[id], nil
}

func (mod *Module) storeValue(addr uint32, v any) error {
	mod.debug("   storeValue(addr=%v type=%T v=%v nil=%v)", addr, v, v, (v == nil))

	// Convert any integer to a float64, which is akin to a JSON number.
	switch num := v.(type) {
	case int:
		v = float64(num)
	case uint:
		v = float64(num)
	case int8:
		v = float64(num)
	case uint8:
		v = float64(num)
	case int16:
		v = float64(num)
	case uint16:
		v = float64(num)
	case int32:
		v = float64(num)
	case uint32:
		v = float64(num)
	case int64:
		v = float64(num)
	case uint64:
		v = float64(num)
	case float32:
		v = float64(num)
	}

	// setNaN sets a NaN-value on the specified address.
	setNaN := func(val uint32) error {
		if err := mod.instance.SetUInt32(addr+4, nanHead); err != nil {
			return err
		}

		return mod.instance.SetUInt32(addr, val)
	}

	// If this is a number, store it as such.
	if num, ok := v.(float64); ok && num != 0 {
		if math.IsNaN(num) {
			return setNaN(0)
		}

		return mod.instance.SetFloat64(addr, num)
	}

	// undefined is stored as a zero float64.
	if v == any(undefined) {
		return mod.instance.SetFloat64(addr, 0)
	}

	// Check for specific values that don't require storing anything in the
	// ids and values map.
	switch v {
	case float64(0):
		return setNaN(1)
	case nil:
		return setNaN(2)
	case true:
		return setNaN(3)
	case false:
		return setNaN(4)
	}

	// Convert slices to the jsArray type.
	if a, ok := v.([]any); ok {
		v = &jsArray{elements: a}
	}

	// Convert strings to the jsString type.
	if str, ok := v.(string); ok {
		v = &jsString{data: str}
	}

	// Convert []byte to the jsUint8Array type.
	if b, ok := v.([]byte); ok {
		v = &jsUint8Array{data: b}
	}

	// Create a unique signature of the value.
	signature := fmt.Sprintf("%d", reflect.ValueOf(v).Pointer())
	mod.debug("   storeValue(type=%T signature=%v)", v, signature)

	// Use the signature to check if this value has already been stored. If not,
	// store it in the ids and values map.
	id, ok := mod.ids[signature]
	if !ok {
		id = mod.getID()
		mod.values[id] = v
		mod.refcounts[id] = 0
		mod.ids[signature] = id
	}

	// Raise the reference count on this object.
	mod.refcounts[id]++

	// Determine if the value needs to be stored with a specific type flag.
	var typeFlag uint32
	switch t := v.(type) {
	case *jsObject, *jsArray, *jsUint8Array, jsProperties:
		if t != nil {
			typeFlag = 1
		}

	case *jsString:
		typeFlag = 2

	case *jsFunction:
		typeFlag = 4

	default:
		panic(fmt.Sprintf("%T: unknown value type", t))
	}

	mod.debug("   storeValue(id=%v typeFlag=%v refcount=%v signature=%q)", id, typeFlag, mod.refcounts[id], signature)

	// Store the type.
	if err := mod.instance.SetUInt32(addr+4, nanHead|typeFlag); err != nil {
		return err
	}

	// Store the ID.
	return mod.instance.SetUInt32(addr, id)
}

// loadSlice returns a byte slice that is referenced by the specified address.
func (mod *Module) loadSlice(addr uint32) ([]byte, error) {
	offset, err := mod.instance.GetInt64(addr)
	if err != nil {
		return nil, err
	}

	length, err := mod.instance.GetInt64(addr + 8)
	if err != nil {
		return nil, err
	}

	mod.debug("   loadSlice(offset=%v length=%v)", offset, length)

	return mod.instance.Range(uint32(offset), uint32(length))
}

// loadSliceOfValues returns a slice of values that is referenced by the
// specified address.
func (mod *Module) loadSliceOfValues(addr uint32) ([]any, error) {
	offset, err := mod.instance.GetInt64(addr)
	if err != nil {
		return nil, err
	}

	length, err := mod.instance.GetInt64(addr + 8)
	if err != nil {
		return nil, err
	}

	a := make([]any, length)
	for i := int64(0); i < length; i++ {
		a[i], err = mod.loadValue(uint32(offset + (i * 8)))
		if err != nil {
			return nil, err
		}
	}

	return a, nil
}

// loadString returns a string that is referenced by the specified address.
func (mod *Module) loadString(addr uint32) (string, error) {
	d, err := mod.loadSlice(addr)
	if err != nil {
		return "", err
	}

	return string(d), nil
}

func (mod *Module) reflectApply(v any, name string, args []any) (any, error) {
	mod.debug("   reflectApply(name=%v)", name)

	obj, err := mod.reflectGet(v, name)
	if err != nil {
		return nil, err
	}

	return mod.reflectConstruct(obj, args)
}

func (mod *Module) reflectConstruct(v any, args []any) (any, error) {
	mod.debug("   reflectConstruct(v=%v args=%v)", v, args)

	if fn, ok := v.(*jsFunction); ok {
		return fn.fn(args), nil
	}

	return nil, fmt.Errorf("%T: not a function", v)
}

func (mod *Module) reflectGet(v, key any) (any, error) {
	mod.debug("   reflectGet(key=%v)", key)

	if v == nil {
		v = mod.values[5]
	}

	if name, ok := key.(string); ok {
		var props jsProperties
		switch vv := v.(type) {
		case *jsObject:
			props = vv.properties
		case jsProperties:
			props = vv
		}
		if props != nil {
			if val, ok := props[name]; ok {
				return val, nil
			}
			return undefined, nil
		}
	}

	index, ok := key.(int64)
	if !ok {
		return nil, errors.New("key not an int64")
	}

	a, ok := v.(*jsArray)
	switch {
	case !ok:
		return nil, errors.New("value not a slice")
	case index < 0 || index >= int64(len(a.elements)):
		return nil, errors.New("index out of range")
	}

	return a.elements[index], nil
}

func (mod *Module) reflectSet(v, key, value any) error {
	mod.debug("   reflectSet(v=%v key=%v value=%v)", v, key, value)

	if v == nil {
		v = mod.values[5]
	}

	if name, ok := key.(string); ok {
		v.(*jsObject).properties[name] = value
		return nil
	}

	index, ok := key.(int64)
	if !ok {
		return errors.New("key not an int64")
	}

	a, ok := v.(*jsArray)
	switch {
	case !ok:
		return errors.New("value not a slice")
	case index < 0 || index >= int64(len(a.elements)):
		return errors.New("index out of range")
	}

	a.elements[index] = value
	return nil
}

func (mod *Module) reflectDeleteProperty(v, key any) error {
	mod.debug("   reflectDelete(v=%v key=%v)", v, key)

	if v == nil {
		v = mod.values[5]
	}

	if name, ok := key.(string); ok {
		delete(v.(*jsObject).properties, name)
		return nil
	}

	index, ok := key.(int64)
	if !ok {
		return errors.New("key not an int64")
	}

	a, ok := v.(*jsArray)
	switch {
	case !ok:
		return errors.New("value not a slice")
	case index < 0 || index >= int64(len(a.elements)):
		return errors.New("index out of range")
	}

	copy(a.elements[index:], a.elements[index+1:])
	a.elements[len(a.elements)-1] = nil
	a.elements = a.elements[:len(a.elements)-1]

	return nil
}

func (mod *Module) wrap(name string, fn func() error) error {
	if fn == nil {
		mod.error("%s NOT IMPLEMENTED", name)
		return nil
	}

	if name != "" && name != "runtime.wasmWrite" {
		mod.debug(name)
	}

	if err := fn(); err != nil {
		if name != "" {
			mod.error("%s: %v", name, err)
		}
		return err
	}

	return nil
}

// ****************************************************************************
// ***************************** [ Go JS module ] *****************************
// ****************************************************************************

// WasmExit is called whenever the WASM program exits.
//
// This method is called from the runtime package.
func (mod *Module) WasmExit(sp uint32) {
	_ = mod.wrap("runtime.wasmExit", func() error {
		if mod.exit == nil {
			return nil
		}

		v, err := mod.instance.GetUInt32(sp + 8)
		if err != nil {
			return err
		}

		mod.exit.Exit(int(v))
		return nil
	})
}

// WasmWrite writes data to a file descriptor.
//
// This method is called from the runtime package.
func (mod *Module) WasmWrite(sp uint32) {
	_ = mod.wrap("runtime.wasmWrite", func() error {
		fd, err := mod.instance.GetInt64(sp + 8)
		if err != nil {
			return err
		}

		p, err := mod.instance.GetInt64(sp + 16)
		if err != nil {
			return err
		}

		n, err := mod.instance.GetUInt32(sp + 24)
		if err != nil {
			return err
		}

		data, err := mod.instance.Range(uint32(p), n)
		if err != nil {
			return err
		}

		_, err = mod.write(int(fd), data)
		return err
	})
}

// ResetMemoryDataView is called whenever WebAssembly's memory.grow instruction
// has been used.
//
// This method is called from the runtime package.
func (mod *Module) ResetMemoryDataView(sp uint32) {
	// Nothing to do: the Instance's Memory is expected to always reflect the
	// current state of the module's memory.
}

// Nanotime1 returns the current time in nanoseconds.
//
// This method is called from the runtime package.
func (mod *Module) Nanotime1(sp uint32) {
	_ = mod.wrap("runtime.nanotime1", func() error {
		return mod.instance.SetInt64(sp+8, time.Now().UnixNano())
	})
}

// Walltime returns the current seconds and nanoseconds.
//
// This method is called from the runtime package.
func (mod *Module) Walltime(sp uint32) {
	_ = mod.wrap("runtime.walltime", func() error {
		msec := time.Now().UnixNano() / int64(time.Millisecond)

		if err := mod.instance.SetInt64(sp+8, msec/1000); err != nil {
			return err
		}

		return mod.instance.SetUInt32(sp+16, uint32(msec%1000)*1000000)
	})
}

// ScheduleTimeoutEvent is called whenever an event needs to be scheduled after
// a certain amount of milliseconds.
//
// This method is called from the runtime package.
func (mod *Module) ScheduleTimeoutEvent(sp uint32) {
	_ = mod.wrap("runtime.scheduleTimeoutEvent", nil)
}

// ClearTimeoutEvent clears a timeout event scheduled by ScheduleTimeoutEvent.
//
// This method is called from the runtime package.
func (mod *Module) ClearTimeoutEvent(sp uint32) {
	_ = mod.wrap("runtime.clearTimeoutEvent", nil)
}

// GetRandomData returns random data.
//
// This method is called from the runtime package.
func (mod *Module) GetRandomData(sp uint32) {
	_ = mod.wrap("runtime.getRandomData", func() error {
		data, err := mod.loadSlice(sp + 8)
		if err != nil {
			return err
		}

		_, err = rand.Read(data)
		return err
	})
}

// FinalizeRef removes a value from memory.
//
// This method is called from various places in syscall/js.Value.
func (mod *Module) FinalizeRef(sp uint32) {
	_ = mod.wrap("syscall/js.finalizeRef", func() error {
		id, err := mod.instance.GetUInt32(sp + 8)
		if err != nil {
			return err
		}

		mod.debug("   id=%v (%T)", id, id)

		// Make sure the ID has a reference count.
		ref, ok := mod.refcounts[id]
		if !ok {
			return fmt.Errorf("%d: missing reference count for id", id)
		}

		mod.debug("   ref=%v", ref)

		// Decrease the reference count.
		ref--

		// If the reference count is 0, clean up the object.
		if ref == 0 {
			mod.debug("   deleting reference")

			signature, ok := mod.values[id]
			if !ok {
				return fmt.Errorf("%d: could not find signature in values for id", id)
			}

			mod.debug("%d: deleting object", id)

			delete(mod.refcounts, id)
			delete(mod.values, id)
			delete(mod.ids, signature)
		} else {
			mod.debug("   decreasing ref")
			mod.refcounts[id] = ref
		}

		return nil
	})
}

// StringVal stores a value as a string.
//
// This method is called from syscall/js.ValueOf().
func (mod *Module) StringVal(sp uint32) {
	_ = mod.wrap("syscall/js.stringVal", func() error {
		v, err := mod.loadString(sp + 8)
		if err != nil {
			return err
		}

		return mod.storeValue(sp+24, &jsString{v})
	})
}

// ValueGet returns the JavaScript property of an object.
//
// This method is called from syscall/js.Value.Get().
func (mod *Module) ValueGet(sp uint32) {
	_ = mod.wrap("syscall/js.valueGet", func() error {
		// Fetch the object.
		v, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		// Fetch the name of the property to read.
		name, err := mod.loadString(sp + 16)
		if err != nil {
			return err
		}

		// Read the value from the property on the object.
		result, err := mod.reflectGet(v, name)
		if err != nil {
			return err
		}

		resultSP, err := mod.instance.GetSP()
		if err != nil {
			return err
		}

		// Store the results.
		return mod.storeValue(resultSP+32, result)
	})
}

// ValueSet sets a value on a property on an object.
//
// This method is called from syscall/js.Value.Set().
func (mod *Module) ValueSet(sp uint32) {
	_ = mod.wrap("syscall/js.valueSet", func() error {
		v, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		name, err := mod.loadString(sp + 16)
		if err != nil {
			return err
		}

		value, err := mod.loadValue(sp + 32)
		if err != nil {
			return err
		}

		return mod.reflectSet(v, name, value)
	})
}

// ValueDelete deletes a property on an object.
//
// This method is called from syscall/js.Value.Delete().
func (mod *Module) ValueDelete(sp uint32) {
	_ = mod.wrap("syscall/js.valueDelete", func() error {
		// Fetch the object.
		v, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		// Fetch the property name.
		name, err := mod.loadString(sp + 16)
		if err != nil {
			return err
		}

		// Delete the property on the object.
		return mod.reflectDeleteProperty(v, name)
	})
}

// ValueIndex returns a value at a particular index in an array.
//
// This method is called from syscall/js.Value.Index().
func (mod *Module) ValueIndex(sp uint32) {
	_ = mod.wrap("syscall/js.valueIndex", func() error {
		// Fetch the object.
		obj, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		// Fetch the index to fetch the value for.
		index, err := mod.instance.GetInt64(sp + 16)
		if err != nil {
			return err
		}

		// Fetch the value on the index in the array.
		result, err := mod.reflectGet(obj, index)
		if err != nil {
			return err
		}

		// Return the value.
		return mod.storeValue(sp+24, result)
	})
}

// ValueSetIndex sets the value at a particular index of an array.
//
// This method is called from syscall/js.Value.SetIndex().
func (mod *Module) ValueSetIndex(sp uint32) {
	_ = mod.wrap("syscall/js.valueSetIndex", func() error {
		// Fetch the object.
		obj, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		// Fetch the index.
		index, err := mod.instance.GetInt64(sp + 16)
		if err != nil {
			return err
		}

		// Fetch the value that should be set on the index in the array.
		value, err := mod.loadValue(sp + 24)
		if err != nil {
			return err
		}

		// Set the value on the index in the array.
		return mod.reflectSet(obj, index, value)
	})
}

// ValueCall calls the method on an object with the give arguments.
//
// This method is called from syscall/js.Value.Call().
func (mod *Module) ValueCall(sp uint32) {
	var resultSP uint32
	err := mod.wrap("syscall/js.valueCall", func() error {
		var err error
		resultSP, err = mod.instance.GetSP()
		if err != nil {
			return err
		}

		// Fetch the object.
		v, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		// Fetch the name of the method to call.
		name, err := mod.loadString(sp + 16)
		if err != nil {
			return err
		}

		// Fetch the arguments to call the method with.
		args, err := mod.loadSliceOfValues(sp + 32)
		if err != nil {
			return err
		}

		// Call the method on the object with the arguments.
		result, err := mod.reflectApply(v, name, args)
		if err != nil {
			return err
		}

		// Store the results of the call.
		if err = mod.storeValue(resultSP+56, result); err != nil {
			return err
		}

		return mod.instance.SetUInt8(resultSP+64, 1)
	})
	if err == nil || resultSP == 0 {
		return
	}

	if err = mod.storeValue(resultSP+56, err.Error()); err != nil {
		return
	}

	_ = mod.instance.SetUInt8(resultSP+54, 0)
}

// ValueInvoke calls the value v with the specified arguments.
//
// This method is called from syscall/js.Value.Invoke().
func (mod *Module) ValueInvoke(sp uint32) {
	var resultSP uint32
	err := mod.wrap("syscall/js.valueInvoke", func() error {
		var err error
		resultSP, err = mod.instance.GetSP()
		if err != nil {
			return err
		}

		// Fetch the function.
		v, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		// Fetch the arguments.
		args, err := mod.loadSliceOfValues(sp + 16)
		if err != nil {
			return err
		}

		// Call v with the specified arguments.
		result, err := mod.reflectConstruct(v, args)
		if err != nil {
			return err
		}

		// Store the results of the call.
		if err = mod.storeValue(resultSP+40, result); err != nil {
			return err
		}

		return mod.instance.SetUInt8(resultSP+48, 1)
	})
	if err == nil || resultSP == 0 {
		return
	}

	if err = mod.storeValue(resultSP+40, err); err != nil {
		return
	}

	_ = mod.instance.SetUInt8(resultSP+48, 0)
}

// ValueNew calls a constructor function with the given arguments. This is akin
// to JavaScript's "new" operator.
//
// This method is called from syscall/js.Value.New().
func (mod *Module) ValueNew(sp uint32) {
	var resultSP uint32
	err := mod.wrap("syscall/js.valueNew", func() error {
		var err error
		resultSP, err = mod.instance.GetSP()
		if err != nil {
			return err
		}

		// Fetch the constructor function.
		v, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		mod.debug("   v: %T: %v", v, v)

		// Fetch the arguments to call the constructor with.
		args, err := mod.loadSliceOfValues(sp + 16)
		if err != nil {
			return err
		}

		mod.debug("   args: %T: %v", args, args)

		// Call the constructor function with the arguments.
		result, err := mod.reflectConstruct(v, args)
		if err != nil {
			return err
		}

		// Store the results of the call.
		if err = mod.storeValue(resultSP+40, result); err != nil {
			return err
		}

		return mod.instance.SetUInt8(resultSP+48, 1)
	})
	if err == nil || resultSP == 0 {
		return
	}

	if err = mod.storeValue(resultSP+40, err); err != nil {
		return
	}

	_ = mod.instance.SetUInt8(resultSP+48, 1)
}

// ValueLength returns the JavaScript property of "length" of v.
//
// This method is called from syscall/js.Value.Length().
func (mod *Module) ValueLength(sp uint32) {
	_ = mod.wrap("syscall/js.valueLength", func() error {
		v, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		switch val := v.(type) {
		case *jsArray:
			return mod.instance.SetInt64(sp+16, int64(len(val.elements)))
		case *jsUint8Array:
			return mod.instance.SetInt64(sp+16, int64(len(val.data)))
		case *jsString:
			return mod.instance.SetInt64(sp+16, int64(len(val.data)))
		default:
			return fmt.Errorf("%T: unknown type for valueLength", v)
		}
	})
}

// ValuePrepareString converts a value to a string and stores it.
//
// This method is called from syscall/js.Value.jsString() for jsString, Boolean
// and Number types.
func (mod *Module) ValuePrepareString(sp uint32) {
	_ = mod.wrap("syscall/js.valuePrepareString", func() error {
		mod.debug("   valuePrepareString: sp=%v", sp)

		v, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		var s *jsString
		switch vv := v.(type) {
		// Boolean.
		case bool:
			s = &jsString{data: fmt.Sprintf("%t", vv)}
		// Number.
		case float64:
			s = &jsString{data: strconv.FormatFloat(vv, 'f', -1, 64)}
		// jsString.
		case *jsString:
			s = vv
		default:
			return fmt.Errorf("%T: unable to convert type to string", v)
		}

		if err = mod.storeValue(sp+16, s); err != nil {
			return err
		}

		return mod.instance.SetInt64(sp+24, int64(len(s.data)))
	})
}

// ValueLoadString loads a string into memory.
//
// This method is called from syscall/js.Value.jsString().
func (mod *Module) ValueLoadString(sp uint32) {
	_ = mod.wrap("syscall/js.valueLoadString", func() error {
		v, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		s, ok := v.(*jsString)
		if !ok {
			return fmt.Errorf("%T: type not a string", v)
		}

		dst, err := mod.loadSlice(sp + 16)
		if err != nil {
			return err
		}

		mod.debug("   string=%v", s.data)

		copy(dst, s.data)
		return nil
	})
}

// ValueInstanceOf returns true when v is an instance of type t.
//
// This method is called from syscall/js.Value.InstanceOf().
func (mod *Module) ValueInstanceOf(sp uint32) {
	_ = mod.wrap("syscall/js.valueInstanceOf", func() error {
		v, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		t, err := mod.loadValue(sp + 16)
		if err != nil {
			return err
		}

		lookup, ok := t.(interface{ Name() string })
		if !ok {
			return mod.instance.SetUInt8(sp+24, 0)
		}

		name := lookup.Name()
		switch v.(type) {
		case *jsArray:
			if name == "Array" {
				return mod.instance.SetUInt8(sp+24, 1)
			}
		case *jsObject:
			if name == "Object" {
				return mod.instance.SetUInt8(sp+24, 1)
			}
		case *jsUint8Array:
			if name == "Uint8Array" {
				return mod.instance.SetUInt8(sp+24, 1)
			}
		}

		return mod.instance.SetUInt8(sp+24, 0)
	})
}

// CopyBytesToGo copies bytes from JavaScript to Go.
func (mod *Module) CopyBytesToGo(sp uint32) {
	_ = mod.wrap("syscall/js.copyBytesToGo", func() error {
		dst, err := mod.loadSlice(sp + 8)
		if err != nil {
			return err
		}

		v, err := mod.loadValue(sp + 32)
		if err != nil {
			return err
		}

		src, ok := v.(*jsUint8Array)
		if !ok {
			return fmt.Errorf("src: %T not type jsUint8Array", v)
		}

		if len(dst) == 0 || len(src.data) == 0 {
			return mod.instance.SetUInt8(sp+48, 0)
		}

		n := copy(dst, src.data)
		if err = mod.instance.SetInt64(sp+40, int64(n)); err != nil {
			return err
		}

		return mod.instance.SetUInt8(sp+48, 1)
	})
}

// CopyBytesToJS copies bytes from Go to JavaScript.
func (mod *Module) CopyBytesToJS(sp uint32) {
	_ = mod.wrap("syscall/js.copyBytesToJS", func() error {
		v, err := mod.loadValue(sp + 8)
		if err != nil {
			return err
		}

		dst, ok := v.(*jsUint8Array)
		if !ok {
			_ = mod.instance.SetUInt8(sp+48, 1)
			return fmt.Errorf("dst: %T not type jsUint8Array", v)
		}

		src, err := mod.loadSlice(sp + 16)
		if err != nil {
			return err
		}

		n := copy(dst.data, src)
		if err = mod.instance.SetInt64(sp+40, int64(n)); err != nil {
			return err
		}

		return mod.instance.SetUInt8(sp+48, 1)
	})
}

// Debug prints some debugging information ... I guess?
func (mod *Module) Debug(sp uint32) {
	_ = mod.wrap("debug", nil)
}
