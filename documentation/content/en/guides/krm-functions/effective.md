---
title: Effective KRM functions
linkTitle: Effective KRM functions
description: Practical patterns for selecting, reading, and updating resources in a KRM function.
toc_hide: false
menu:
  main:
    parent: "KRM Function Developer Guide"
    weight: 25
---
This guide covers patterns for writing KRM functions that go beyond the basic
tutorial. Read [Tutorial]({{% relref "/guides/krm-functions/tutorial" %}}) and
[Interfaces]({{% relref "/guides/krm-functions/interfaces" %}}) first.

The examples use the [KRM function SDK](https://pkg.go.dev/github.com/kptdev/krm-functions-sdk/go/fn).

## Prerequisites

- [Install kpt]({{% relref "/installation" %}})
- [Install Docker](https://docs.docker.com/get-docker/)

## Read configuration

The function config arrives as a `KubeObject` in the `functionConfig` argument of
`Run`. Read the fields you need from it instead of defining a struct for them.

```golang
func (r *SetImage) Run(ctx *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects, results *fn.Results) bool {
	oldImage, _, err := functionConfig.NestedString("oldImage")
	if err != nil {
		results.ErrorE(err)
		return false
	}
	newImage, _, err := functionConfig.NestedString("newImage")
	if err != nil {
		results.ErrorE(err)
		return false
	}
	// ... use oldImage and newImage
	return true
}
```

In `main`, wrap your `Runner` with `fn.WithContext` before passing it to
`fn.AsMain`, as shown in [Interfaces]({{% relref "/guides/krm-functions/interfaces" %}}).
Passing it directly fails at runtime with `unknown input type`.

A matching input looks like this:

```yaml
apiVersion: fn.kpt.dev/v1alpha1
kind: SetImage
metadata:
  name: try-out
oldImage: example
newImage: my-registry/example:v2
```

## Select KRM resources

`fn.KubeObjects` is a slice of `*fn.KubeObject`. Use `Where` and `WhereNot` with
a predicate to narrow the set. The `GroupVersionKind` and `GroupKind` types come from
`github.com/kptdev/kpt/api/schema/v1`, not from `k8s.io/apimachinery`.

```golang
import schema "github.com/kptdev/kpt/api/schema/v1"

// Namespace-scoped resources only.
namespaced := items.Where(func(o *fn.KubeObject) bool {
	return o.IsNamespaceScoped()
})

// Cluster-scoped resources only.
clusterScoped := items.Where(func(o *fn.KubeObject) bool {
	return o.IsClusterScoped()
})

// Deployments in the apps group, at exactly apps/v1.
deployments := items.Where(fn.IsGroupVersionKind(schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "Deployment",
}))

// Everything except Namespace objects, in the core group.
excluded := items.WhereNot(fn.IsGroupKind(schema.GroupKind{
	Group: "",
	Kind:  "Namespace",
}))
```

Use `fn.IsGroupVersionKind` to match an exact apiVersion and kind. Use
`fn.IsGroupKind` to match a group and kind across all versions. `IsGVK` is
deprecated, so avoid it in new code.

`IsNamespaceScoped` is decided by a table of known kinds. For any other kind it
falls back to whether `metadata.namespace` is set. A custom resource without a
namespace is treated as cluster-scoped, so check the kind if this matters for
your function.

## Read and update fields

`KubeObject` embeds `SubObject`, so both expose the same field accessors. Each
nested getter returns the value, whether the field was found, and an error.

```golang
for _, obj := range items.Where(fn.IsGroupVersionKind(schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "Deployment",
})) {
	replicas, found, err := obj.NestedInt64("spec", "replicas")
	if err != nil {
		results.ErrorE(err)
		continue
	}
	if found {
		fn.Logf("%s has %d replicas\n", obj.GetName(), replicas)
	}

	strategy, _, err := obj.NestedString("spec", "strategy", "type")
	if err != nil {
		results.ErrorE(err)
		continue
	}
	if strategy == "Recreate" {
		if err := obj.SetNestedString("RollingUpdate", "spec", "strategy", "type"); err != nil {
			results.ErrorE(err)
		}
	}
}
```

Setters return an error. Check it, as shown above.

## Work with sub-objects

`GetMap` returns a `SubObject` for a nested map. If the field is missing, it
returns `nil`, and calling a method on `nil` panics. Check each step before you
chain the next one.

```golang
spec := deployment.GetMap("spec")
if spec == nil {
	return true // no spec, nothing to update
}
template := spec.GetMap("template")
if template == nil {
	return true
}
podSpec := template.GetMap("spec")
if podSpec == nil {
	return true
}

if err := podSpec.SetNestedString("ssd", "nodeSelector", "disktype"); err != nil {
	results.ErrorE(err)
}
```

## Decode into a typed struct

When a resource has a shape you already model in Go, decode its sub-object into a
struct with `As`. `As` returns an error on a `nil` sub-object, so it is safe to
call without a check, but you still need to handle that error.

```golang
type deploymentSpec struct {
	Replicas int64 `json:"replicas"`
}

var spec deploymentSpec
if err := deployment.GetMap("spec").As(&spec); err != nil {
	results.ErrorE(err)
	return false
}
fn.Logf("decoded %d replicas\n", spec.Replicas)
```

## Next steps

- Read [Testing]({{% relref "/guides/krm-functions/testing" %}}) to cover your
  selection and update logic with golden tests.
- Read [Containerizing]({{% relref "/guides/krm-functions/containerizing" %}}) to
  package the function as an image.
