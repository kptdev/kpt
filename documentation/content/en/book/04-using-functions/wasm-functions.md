---
title: "Using WASM Functions"
linkTitle: "Using WASM Functions"
weight: 5
description: >
    How to run, develop, and deploy WebAssembly (WASM) functions with kpt.
---

WASM functions are an alternative to container-based KRM functions. They're faster to start, smaller to distribute, and run in a sandboxed environment.

## Why use WASM functions?

{{< warning type=warning title="Note" >}}
WASM support in kpt is currently in alpha and not ready for production use. The API and behavior may change in future releases.
{{< /warning >}}

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
    - image: example.registry.io/my-org/my-wasm-fn:v1.0.0
      configMap:
        key: value
```

```shell
kpt fn render my-package --allow-alpha-wasm
```

kpt will detect the function as WASM if the OCI image has a `js/wasm` platform manifest.

### With `fn eval`

Run WASM functions imperatively:

```shell
kpt fn eval my-package --allow-alpha-wasm -i example.registry.io/my-org/my-wasm-fn:v1.0.0 -- key=value
```

### Using local WASM files

You can run local `.wasm` files with the `--exec` flag:

```shell
kpt fn eval my-package --allow-alpha-wasm --allow-exec --exec ./my-function.wasm
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
kpt fn render my-package --allow-alpha-wasm --allow-exec
```

Note: Using local WASM files makes your package less portable since the file needs to exist on every system.

## Publishing WASM functions

### Push a WASM module

Compress and push a WASM module to an OCI registry:

```shell
kpt alpha wasm push ./my-function.wasm example.registry.io/my-org/my-wasm-fn:v1.0.0
```

What this does:
1. Compresses the WASM file into a tar archive
2. Creates an OCI image with `js/wasm` platform
3. Pushes to the registry

### Pull a WASM module

Download and decompress a WASM module:

```shell
kpt alpha wasm pull example.registry.io/my-org/my-wasm-fn:v1.0.0 ./my-function.wasm
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

For a complete working example of a Go WASM function, see the [starlark function in the krm-functions-catalog](https://github.com/kptdev/krm-functions-catalog/tree/main/functions/go/starlark), which shows the two-file pattern (`run.go` for regular builds and `run_js.go` for WASM builds).

### Build for WASM

```shell
GOOS=js GOARCH=wasm go build -o my-function.wasm .
```

### Test locally

```shell
kpt fn eval ./test-package --allow-alpha-wasm --allow-exec --exec ./my-function.wasm
```

### Publish

Push to a registry:

```shell
kpt alpha wasm push ./my-function.wasm example.registry.io/my-org/my-wasm-fn:v1.0.0

# Test the published version
kpt fn eval ./test-package --allow-alpha-wasm -i example.registry.io/my-org/my-wasm-fn:v1.0.0
```

## Complete example

From development to deployment:

```shell
# 1. Write your function code (main.go and main_js.go)

# 2. Build
GOOS=js GOARCH=wasm go build -o my-function.wasm .

# 3. Test locally
kpt fn eval ./test-package --allow-alpha-wasm --allow-exec --exec ./my-function.wasm

# 4. Publish
kpt alpha wasm push ./my-function.wasm example.registry.io/my-org/my-wasm-fn:v1.0.0

# 5. Use in a package
cat <<EOF > Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-package
pipeline:
  mutators:
    - image: example.registry.io/my-org/my-wasm-fn:v1.0.0
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
