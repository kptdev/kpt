You can write the function in Starlark script.

?> Starlark SDK is in *experimental* stage.

[Starlark] is a python-like language designed for use in configuration files that has several desirable properties:
* deterministic evaluation
* hermetic execution 
* simplicity.

Current Starlark SDK is driven by [`gcr.io/kpt-fn/starlark:v0.4`] which contains the interpreter and accepts 
a `StarlarkRun` object as its `FunctionConfig`. You should place your starlark script in the `source` field
of the `StarlarkRun` object. 

## Quickstart

Let's write a starlark function which add annotation "managed-by=kpt" only to `Deployment` resources.

### Get the "get-started" example

```shell
kpt pkg get https://github.com/GoogleContainerTools/kpt-functions-sdk.git/starlark/get-started@master set-annotation
cd set-annotation
```

### Update the `FunctionConfig`
```yaml
# starlark-fn-config.yaml
apiVersion: fn.kpt.dev/v1alpha1
kind: StarlarkRun
metadata:
  name: set-annotation
# EDIT THE SOURCE! 
# This should be your starlark script which preloads the `ResourceList` to `ctx.resource_list`
source: |
  for resource in ctx.resource_list["items"]:
    if resource.get("kind") == "Deployment":
      resource["metadata"]["annotations"]["managed-by"] = "kpt"
```
In the `source` field, the `ResourceList` from STDIN is loaded to `ctx.resource_list` as a dict. 
You can manipulate KRM resource as operating on a dict.

### Test and Run

Run the starlark script via `kpt`   
```shell
# `starlark:v0.4` is the short form of gcr.io/kpt-fn/starlark:v0.4 catalog function. 
kpt fn eval ./data --image starlark:v0.4 --fn-config starlark-fn-config.yaml

# Verify that the annotation is added to the `Deployment` resource and the other resource `Service` 
# does not have this annotation.
cat ./data/resources.yaml | grep annotations -A1 -B5
```

?> Refer to the [Functions Catalog](https://catalog.kpt.dev/starlark/v0.4/) for
details on how to use this function.

[`gcr.io/kpt-fn/starlark:v0.4`]: https://catalog.kpt.dev/starlark/v0.4/
[Starlark]: https://github.com/bazelbuild/starlark#starlark