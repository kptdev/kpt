---
title: Tutorial
linkTitle: Tutorial
description: Build a KRM function end to end with the Go SDK.
toc_hide: false
menu:
  main:
    parent: "KRM Function Developer Guide"
    weight: 10
---
This tutorial walks through the end-to-end workflow for building a KRM function
using the Go SDK. By the end, you will have a working function with embedded
documentation, golden tests, and support for `--help`, `--doc`, and standalone
file mode.

For a complete working example, see [`go/get-started/`](https://github.com/kptdev/krm-functions-sdk/tree/main/go/get-started).

## 1. Create Your Function

A KRM function implements the `fn.Runner` interface:

```go
type Runner interface {
    Run(context *Context, functionConfig *KubeObject, items KubeObjects, results *Results) bool
}
```

Here is a minimal function that sets labels on all the resources:

```go
package main

import (
    "context"
    _ "embed"
    "os"

    "github.com/kptdev/krm-functions-sdk/go/fn"
)

//go:embed README.md
var readme []byte

//go:embed metadata.yaml
var metadata []byte

var _ fn.Runner = &SetLabels{}

type SetLabels struct {
    Labels map[string]string `json:"labels,omitempty"`
}

func (r *SetLabels) Run(ctx *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects, results *fn.Results) bool {
    for _, obj := range items {
        for k, v := range r.Labels {
            if err := obj.SetLabel(k, v); err != nil {
                results.ErrorE(err)
            }
        }
    }
    return results.ExitCode() == 0
}

func main() {
    runner := fn.WithContext(context.Background(), &SetLabels{})
    if err := fn.AsMain(runner, fn.WithDocs(readme, metadata)); err != nil {
        os.Exit(1)
    }
}
```

Key points:
- Your struct fields are automatically populated from `functionConfig` (JSON unmarshaling).
- Return `true` for success, `false` for failure.
- Use `results` to report structured messages (info, warning, error).

## 2. Embed Documentation with `//go:embed`

The SDK uses Go's embed directive to bundle documentation into the binary.
Two files are needed:

### README.md

Use `<!--mdtogo-->` markers to define sections that `--help` and `--doc` extract:

    # set-labels

    <!--mdtogo:Short-->
    Set labels on all resources in the package.
    <!--mdtogo-->

    <!--mdtogo:Long-->
    ## Usage

    The `set-labels` function adds or updates labels on all KRM resources.
    It accepts a `SetLabels` functionConfig with a `labels` map.

    ### FunctionConfig

    ```yaml
    apiVersion: fn.kpt.dev/v1alpha1
    kind: SetLabels
    metadata:
      name: my-config
    labels:
      app: my-app
      env: production
    ```

    <!--mdtogo-->

    <!--mdtogo:Examples-->

    Set a single label on all resources:

    ```yaml
    apiVersion: fn.kpt.dev/v1alpha1
    kind: SetLabels
    labels:
      team: platform
    ```

    <!--mdtogo-->

### metadata.yaml

```yaml
image: ghcr.io/kptdev/krm-functions-catalog/set-labels:v0.1
description: Set labels on all resources
tags:
  - mutator
  - labels
sourceURL: https://github.com/kptdev/krm-functions-catalog/tree/main/functions/go/set-labels
examplePackageURLs:
  - https://github.com/kptdev/krm-functions-catalog/tree/main/examples/set-labels-simple
license: Apache-2.0
hidden: false
```

### Wire it up

In your `main.go`:

```go
//go:embed README.md
var readme []byte

//go:embed metadata.yaml
var metadata []byte

func main() {
    runner := fn.WithContext(context.Background(), &SetLabels{})
    if err := fn.AsMain(runner, fn.WithDocs(readme, metadata)); err != nil {
        os.Exit(1)
    }
}
```

## 3. Running Your Function

### Standard mode (STDIN/STDOUT)

Pipe a ResourceList through your function:

```bash
cat input.yaml | go run . > output.yaml
```

### Help mode

View human-readable documentation:

```bash
go run . --help
```

This prints the Short, Long, and Examples sections extracted from your README markers.

### Doc mode

Get machine-readable JSON documentation (consumed by `kpt fn doc` and catalog pipelines):

```bash
go run . --doc
```

### File mode

Process KRM files directly without constructing a ResourceList:

```bash
go run . deployment.yaml service.yaml
```

This reads the YAML files, assembles them into a ResourceList with an empty
functionConfig, processes them, and writes the result to STDOUT.

## 4. Testing with Golden Tests

The SDK provides `testhelpers.RunGoldenTests` for snapshot-based testing.

Create a test directory structure:

```
testdata/
├── test-case-1/
│   ├── _expected.yaml    # Expected output (ResourceList YAML)
│   ├── _fnconfig.yaml    # FunctionConfig for this test case
│   └── resources.yaml    # Input resources
└── test-case-2/
    ├── _expected.yaml
    ├── _fnconfig.yaml
    └── resources.yaml
```

Write your test:

```go
func TestFunction(t *testing.T) {
    runner := fn.WithContext(context.TODO(), &SetLabels{})
    testhelpers.RunGoldenTests(t, "testdata", runner)
}
```

Update expected output after changes:

```bash
WRITE_GOLDEN_OUTPUT=1 go test ./...
```

See [Testing]({{% relref "/guides/krm-functions/testing" %}}) for more details.

## 5. Next Steps

- [Interfaces]({{% relref "/guides/krm-functions/interfaces" %}}) — when to use `fn.Runner` vs `fn.ResourceListProcessor`
- [Testing]({{% relref "/guides/krm-functions/testing" %}}) — golden test patterns in depth
- [Containerizing]({{% relref "/guides/krm-functions/containerizing" %}}) — packaging your function as a container image
