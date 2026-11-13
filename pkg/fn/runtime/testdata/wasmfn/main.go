// Copyright 2026 The kpt Authors
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

// This program is compiled with GOOS=js GOARCH=wasm by the tests in the
// runtime package. It mimics how KRM functions are exposed to kpt: it registers
// processResourceList and processResourceListErrors as JavaScript globals and
// then keeps running so that the host can call them.
package main

import (
	"fmt"
	"strings"
	"syscall/js"
)

func main() {
	// Writing to stdout goes through the fs.write import of the host.
	fmt.Println("wasmfn started")

	js.Global().Set("processResourceList", js.FuncOf(func(_ js.Value, args []js.Value) any {
		input := args[0].String()
		// Output that is not a ResourceList makes the host ask for the errors.
		if strings.Contains(input, "return-invalid") {
			return "kind: NotAResourceList\n"
		}
		// Writing to stdout from inside the callback currently fails in the
		// host, see TestWazeroFnPrintInsideCallback.
		if strings.Contains(input, "print-in-callback") {
			fmt.Println("printing from inside processResourceList")
		}
		return input
	}))

	js.Global().Set("processResourceListErrors", js.FuncOf(func(_ js.Value, args []js.Value) any {
		return "guest error details"
	}))

	<-make(chan bool)
}
