---
title: "Starlark Runtime"
linkTitle: "Starlark Runtime"
weight: 4
type: docs
description: >
   Writing functions as containers
---

{{% pageinfo color="warning" %}}
The Starlark runtime is Alpha, and must be enabled with the `--enable-star`
flag.
{{% /pageinfo %}}

Functions may be written as Starlark scripts which modify a ResourceList provided as
a variable.

#### Imperative Run

Starlark functions can be run imperatively by specifying them in the functionConfig.  Following
is an example of a Starlark function which adds a "foo" annotation to each resource in its
package.

```python
# c.star
# set the foo annotation on each resource
def run(r, an):
  for resource in r:
    resource["metadata"]["annotations"]["foo"] = an

an = ctx.resource_list["functionConfig"]["data"]["value"]
run(ctx.resource_list["items"], an)
```

Run the Starlark function with:

```
# run c.star as a function, generating a ConfigMap with value=bar as the functionConfig
kpt fn run . --enable-star --star-path c.star -- value=bar
```

Any resource in the directory will have the `foo: bar` annotation added.

#### Declarative Run

Starlark functions can be run declaratively using the `config.kubernetes.io/function`
annotation.  Following is an example of a Starlark function which adds a "foo" annotation to
each resource in its package.

```yaml
apiVersion: example.com/v1beta1
kind: ExampleKind
metadata:
  name: function-input
  annotations:
    config.kubernetes.io/function: |
      starlark: {path: c.star, name: example-name}
spec:
  value: "hello world"
```

Example Starlark function to add an annotation:

```python
# c.star
# set the foo annotation on each resource
def run(r, an):
  for resource in r:
    resource["metadata"]["annotations"]["foo"] = an

an = ctx.resource_list["functionConfig"]["spec"]["value"]
run(ctx.resource_list["items"], an)
```

And running them on the directory containing the functionConfig yaml with:

```shell script
kpt fn run DIR/ --enable-star
```

