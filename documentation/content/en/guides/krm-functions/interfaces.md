---
title: Interfaces
linkTitle: Interfaces
description: Choose between fn.Runner and fn.ResourceListProcessor.
toc_hide: false
menu:
  main:
    parent: "KRM Function Developer Guide"
    weight: 20
---
The SDK provides two interfaces for implementing KRM functions. Choose according
to your function requirements.

> The `main` functions below call `fn.AsMain` without `fn.WithDocs` to keep the
> interface examples focused. Production functions should embed documentation and
> pass `fn.WithDocs(readme, metadata)` — see the
> [tutorial]({{% relref "/guides/krm-functions/tutorial" %}}).

## fn.Runner

Use `fn.Runner` for **transformers** (mutators) and **validators**. This is the
recommended interface for most functions.

```go
type Runner interface {
    Run(context *Context, functionConfig *KubeObject, items KubeObjects, results *Results) bool
}
```

Characteristics:
- The SDK automatically parses `functionConfig` into your struct's exported fields.
  A typed functionConfig (its `kind` matching your struct name) is unmarshaled via
  JSON tags; alternatively, a `ConfigMap` functionConfig has its `.data` map assigned
  to a `map[string]string` field on your struct.
- You can **modify** existing items, but adding or removing items is not supported.
  This is a convention, not a compile-time restriction: the SDK does not read back
  items appended inside `Run`, so adds and removes are effectively dropped. Use
  `fn.ResourceListProcessor` when you need to add or remove items.
- Return `true` for success, `false` for failure.
- Use `results` to report structured info/warning/error messages.

### Example: Validator

```go
var _ fn.Runner = &EnforceNamespace{}

type EnforceNamespace struct {
    Namespace string `json:"namespace"`
}

func (r *EnforceNamespace) Run(ctx *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects, results *fn.Results) bool {
    for _, obj := range items {
        if obj.GetNamespace() != r.Namespace {
            results.Errorf("resource %s/%s has namespace %q, expected %q",
                obj.GetKind(), obj.GetName(), obj.GetNamespace(), r.Namespace)
        }
    }
    return results.ExitCode() == 0
}

func main() {
    runner := fn.WithContext(context.Background(), &EnforceNamespace{})
    if err := fn.AsMain(runner); err != nil {
        os.Exit(1)
    }
}
```

### Example: Transformer (Mutator)

```go
var _ fn.Runner = &SetAnnotations{}

type SetAnnotations struct {
    Annotations map[string]string `json:"annotations,omitempty"`
}

func (r *SetAnnotations) Run(ctx *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects, results *fn.Results) bool {
    for _, obj := range items {
        for k, v := range r.Annotations {
            if err := obj.SetAnnotation(k, v); err != nil {
                results.ErrorE(err)
            }
        }
    }
    return results.ExitCode() == 0
}
```

## fn.ResourceListProcessor

Use `fn.ResourceListProcessor` for **generators** and **complex functions** that
need full control over the ResourceList.

```go
type ResourceListProcessor interface {
    Process(rl *ResourceList) (bool, error)
}
```

Characteristics:
- Full access to `ResourceList.Items` — you can add, remove, or modify items.
- You must parse `functionConfig` manually from `rl.FunctionConfig`.
- You can modify `rl.Results` directly.
- Return `(true, nil)` for success, `(false, err)` for failure.

### Example: Generator

```go
type ConfigMapGenerator struct{}

func (g *ConfigMapGenerator) Process(rl *fn.ResourceList) (bool, error) {
    // Parse functionConfig manually
    name, _, _ := rl.FunctionConfig.NestedString("metadata", "name")

    // Generate a new ConfigMap
    cm := fn.NewEmptyKubeObject()
    if err := cm.SetAPIVersion("v1"); err != nil {
        return false, err
    }
    if err := cm.SetKind("ConfigMap"); err != nil {
        return false, err
    }
    if err := cm.SetName(name + "-generated"); err != nil {
        return false, err
    }
    if err := cm.SetNamespace("default"); err != nil {
        return false, err
    }

    // Add to items
    rl.Items = append(rl.Items, cm)
    return true, nil
}

func main() {
    if err := fn.AsMain(&ConfigMapGenerator{}); err != nil {
        os.Exit(1)
    }
}
```

### ResourceListProcessorFunc

For simple cases, use the function adapter instead of defining a struct:

```go
type ResourceListProcessorFunc func(rl *ResourceList) (bool, error)
```

Example:

```go
func main() {
    processor := fn.ResourceListProcessorFunc(func(rl *fn.ResourceList) (bool, error) {
        for _, obj := range rl.Items {
            if err := obj.SetLabel("managed-by", "my-function"); err != nil {
                return false, err
            }
        }
        return true, nil
    })
    if err := fn.AsMain(processor); err != nil {
        os.Exit(1)
    }
}
```

## Choosing Between Interfaces

| Capability | fn.Runner | fn.ResourceListProcessor |
|---|---|---|
| Auto-parse functionConfig | ✅ | ❌ (manual) |
| Modify existing items | ✅ | ✅ |
| Add new items | ❌ | ✅ |
| Remove items | ❌ | ✅ |
| Access full ResourceList | ❌ | ✅ |
| Best for | Transformers, Validators | Generators, Complex functions |

As a rule of thumb, pick the interface by what your function does:

- **Transformers and validators** — use `fn.Runner`. It auto-parses the
  functionConfig and keeps the function focused on modifying items. Examples:
  set-labels, set-namespace.
- **Generators and functions needing full ResourceList access** (adding or
  removing items, reading results from earlier functions) — use
  `fn.ResourceListProcessor`. Examples: render-helm-chart, starlark.

Both produce spec-compliant ResourceList I/O; the choice is about ergonomics, so
use the one that fits your function rather than a hard requirement.

## Wrapping a Runner

`fn.Runner` is wrapped into a `ResourceListProcessor` internally using
`fn.WithContext`:

```go
runner := fn.WithContext(context.Background(), &MyFunction{})
// runner implements ResourceListProcessor and can be passed to fn.AsMain
```

This wrapper handles the following:
1. Parsing `functionConfig` into your struct fields
2. Calling your `Run` method with the parsed context
3. Collecting results and determining success/failure

---

Next: [Testing]({{% relref "/guides/krm-functions/testing" %}}) — golden test patterns for verifying your function.
