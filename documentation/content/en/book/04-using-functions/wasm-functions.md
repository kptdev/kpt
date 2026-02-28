---
title: "Using WASM Functions"
linkTitle: "Using WASM Functions"
weight: 5
description: >
    How to run, develop, and deploy WebAssembly (WASM) functions with kpt.
---

WASM functions are an alternative to container-based KRM functions. They're faster to start, smaller to distribute, and run in a sandboxed environment.

## Why use WASM functions?

WASM functions have some advantages over container-based functions:

- Faster startup - no container runtime needed
- Smaller size - WASM modules are typically much smaller than container images
- Better security - sandboxed execution with no host access by default
- More portable - run anywhere WASM is supported
- Lower resource usage

## Running WASM functions

WASM support is an alpha feature. You need to use the `--allow-alpha-wasm` flag to enable it.

### With `fn render`

Add the WASM function to your Kptfile pipeline:

```yaml
# Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-package
pipeline:
  mutators:
    - image: gcr.io/my-org/my-wasm-fn:v1.0.0
      configMap:
        key: value
```

```shell
kpt fn render my-package --allow-alpha-wasm
```

kpt will detect the function as WASM if the OCI image has a `wasm/js` platform manifest.

### With `fn eval`

Run WASM functions imperatively:

```shell
kpt fn eval my-package --allow-alpha-wasm -i gcr.io/my-org/my-wasm-fn:v1.0.0 -- key=value
```

### Using local WASM files

You can run local `.wasm` files with the `--exec` flag:

```shell
kpt fn eval my-package --allow-alpha-wasm --exec ./my-function.wasm
```

You can also declare local WASM files in your `Kptfile`:

```yaml
# Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-package
pipeline:
  mutators:
    - exec: ./functions/my-function.wasm
```

```shell
kpt fn render my-package --allow-alpha-wasm
```

Note: Using local WASM files makes your package less portable since the file needs to exist on every system.

## Publishing WASM functions

### Push a WASM module

Compress and push a WASM module to an OCI registry:

```shell
kpt alpha wasm push ./my-function.wasm gcr.io/my-org/my-wasm-fn:v1.0.0
```

What this does:
1. Compresses the WASM file into a tar archive
2. Creates an OCI image with `wasm/js` platform
3. Pushes to the registry

### Pull a WASM module

Download and decompress a WASM module:

```shell
kpt alpha wasm pull gcr.io/my-org/my-wasm-fn:v1.0.0 ./my-function.wasm
```

Useful for:
- Testing WASM functions locally
- Inspecting modules
- Offline caching

## Developing WASM functions

WASM functions follow the [KRM Functions Specification](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md). They receive a `ResourceList` as input and return a `ResourceList` as output.

### What you need

- A language that compiles to WASM (Go, Rust, etc.)
- WASM build toolchain
- KRM functions SDK

### Example: Go WASM function

Here's how to build a Go KRM function for WASM. You need two files - one for regular builds and one for WASM:

`main.go` (regular build):

```go
//go:build !(js && wasm)

package main

import (
    "os"
    
    "github.com/kptdev/krm-functions-sdk/go/fn"
)

func main() {
    if err := fn.AsMain(fn.ResourceListProcessorFunc(process)); err != nil {
        os.Exit(1)
    }
}

func process(rl *fn.ResourceList) (bool, error) {
    for i := range rl.Items {
        // Your transformation logic
        rl.Items[i].SetAnnotation("processed-by", "my-fn")
    }
    return true, nil
}
```

`main_js.go` (WASM build):

```go
//go:build js && wasm

package main

import (
    "syscall/js"
    
    "github.com/kptdev/krm-functions-sdk/go/fn"
)

func main() {
    if err := run(); err != nil {
        panic(err)
    }
}

func run() error {
    resourceList := []byte("")
    
    js.Global().Set("processResourceList", resourceListWrapper(&resourceList))
    js.Global().Set("processResourceListErrors", resourceListErrors(&resourceList))
    
    // Keep the program running
    <-make(chan bool)
    return nil
}

func transform(input []byte) ([]byte, error) {
    return fn.Run(fn.ResourceListProcessorFunc(process), input)
}

func resourceListWrapper(resourceList *[]byte) js.Func {
    return js.FuncOf(func(this js.Value, args []js.Value) any {
        if len(args) != 1 {
            return "Invalid number of arguments"
        }
        input := args[0].String()
        output, err := transform([]byte(input))
        *resourceList = output
        if err != nil {
            return "unable to process: " + err.Error()
        }
        return string(output)
    })
}

func resourceListErrors(resourceList *[]byte) js.Func {
    return js.FuncOf(func(this js.Value, args []js.Value) any {
        rl, err := fn.ParseResourceList(*resourceList)
        if err != nil {
            return ""
        }
        errors := ""
        for _, r := range rl.Results {
            if r.Severity == "error" {
                errors += r.Message
            }
        }
        return errors
    })
}
```

### Build for WASM

```shell
GOOS=js GOARCH=wasm go build -o my-function.wasm .
```

### Test locally

```shell
kpt fn eval ./test-package --allow-alpha-wasm --exec ./my-function.wasm
```

### Publish

Push to a registry:

```shell
kpt alpha wasm push ./my-function.wasm gcr.io/my-org/my-wasm-fn:v1.0.0

# Test the published version
kpt fn eval ./test-package --allow-alpha-wasm -i gcr.io/my-org/my-wasm-fn:v1.0.0
```

## Complete example

From development to deployment:

```shell
# 1. Write your function code (main.go and main_js.go)

# 2. Build
GOOS=js GOARCH=wasm go build -o my-function.wasm .

# 3. Test locally
kpt fn eval ./test-package --allow-alpha-wasm --exec ./my-function.wasm

# 4. Publish
kpt alpha wasm push ./my-function.wasm gcr.io/my-org/my-wasm-fn:v1.0.0

# 5. Use in a package
cat <<EOF > Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-package
pipeline:
  mutators:
    - image: gcr.io/my-org/my-wasm-fn:v1.0.0
EOF

# 6. Render
kpt fn render my-package --allow-alpha-wasm
```

## Limitations

### Alpha status

WASM support is alpha, which means:
- API may change
- Requires `--allow-alpha-wasm` flag
- Some features may be limited

### Security

WASM functions run in a sandbox:
- No network access
- No filesystem access (except input/output resources)
- Can't execute system commands

This is more secure but also more restrictive.

### Performance

- Faster startup than containers
- CPU-intensive operations may be slower
- Different memory patterns
- Benchmark if performance is critical

### Compatibility

Not all container functions can convert to WASM:
- Functions needing network access
- Functions with complex dependencies
- Platform-specific code

## See also

- [KRM Functions Specification](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md)
- [Functions Catalog](https://catalog.kpt.dev/)
- [kpt alpha wasm push](../../../reference/cli/alpha/wasm/push/)
- [kpt alpha wasm pull](../../../reference/cli/alpha/wasm/pull/)
- [WASM function examples](https://github.com/kptdev/krm-functions-catalog/tree/main/functions/go)
