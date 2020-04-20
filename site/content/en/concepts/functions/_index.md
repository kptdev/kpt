---
title: "Functions"
linkTitle: "Functions"
weight: 4
type: docs
menu:
  main:
    weight: 5
description: >
   Functions
---

{{% pageinfo color="warning" %}}
# Notice: Under Development
{{% /pageinfo %}}

Kpt functions are conceptually similar to Kubernetes *Controllers*; however they can be run
outside the kubernetes control-plane.  This allows them to be run locally, as part of CICD,
etc.

Kpt functions have a declarative in / declarative out model, similar to Kustomize plugins
(generators / transformers) and Metacontroller.

Kpt functions may be implemented as either Containers or Starlark scripts.
 
- Containers read a `ResourceList` from STDIN and write a `ResourceList` to STDOUT
- Starlark scripts modify the `ctx.resource_list` dictionary variable


**Note:** the [kpt-functions-sdk] provides an SDK for writing functions in typescript.

## ResourceList

Functions work by emitting a modified `ResourceList` which was provided to them as input.

### Items

The ResourceList contains a list of resource items, configuration for the function, and results.

```yaml
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  spec:
  ...
- apiVersion: v1
  kind: Service
  spec:
  ...
```

### FunctionConfig

Functions may be configured using the `functionConfig` field.

```yaml
kind: ResourceList
functionConfig:
  apiVersion: example.com/v1alpha1
  kind: Foo
  spec:
    foo: bar
    ...
items:
  ...
```

Functions may be written to use a ConfigMap as the functionConfig.
When functions are run using the form `kpt fn run DIR/ --image foo:v1 -- a=b c=d`, the arguments
following `--` are parsed into a ConfigMap and set as the functionConfig field provided
as part of the ResourceList.

```yaml
kind: ResourceList
functionConfig:
  apiVersion: v1
  kind: ConfigMap
  spec:
    a: b
    ...
items:
  ...
```

### Results

Functions may define validation results through the `results` field.
When functions are run using the `--results-dir`, each functions results will be written
to a file under the provided directory.

```yaml
kind: ResourceList
functionConfig:
  apiVersion: example.com/v1alpha1
  kind: Foo
  spec:
    foo: bar
    ...
results:
- name: "kubeval"
  items:  
  - severity: error # one of ["error", "warn", "info"] -- error code should be non-0 if there are 1 or more errors
    tags: # arbitrary metadata about the result
      error-type: "field"
    message: "Value exceeds the namespace quota, reduce the value to make the pod schedulable"
    resourceRef: # key to lookup the resource
      apiVersion: apps/v1
      kind: Deployment
      name: foo
      namespace: bar
    file:
      # optional if present as annotation
      path: deploy.yaml # read from annotation if present
      # optional if present as annotation
      index: 0 # read from annotation if present
    field:
      path: "spec.template.spec.containers[3].resources.limits.cpu"
      currentValue: "200" # number | string | boolean
      suggestedValue: "2" # number | string | boolean
  - severity: warn
    ...
- name: "something else"
  items:
  - severity: info
     ...
```

## Running Functions

Functions may be run either imperatively using the form `kpt fn run DIR/ --image fn`, or they
may be run declaratively using the form `kpt fn run DIR/`.


### Imperative Run

To run a specific function against a package, specify that function with the `--image` flag.
If key-value pairs are provided after a `--` argument, then a ConfigMap will be generated
to contain these values as `data` elements, and the ConfigmMap will be set as the functionConfig
field.

The package will be read as input, provided to the function through the `ResourceList.items` field,
and the function output will be written back to the package if there are no errors.

```sh
kpt pkg get https://github.com/GoogleContainerTools/kpt-functions-sdk.git/example-configs example-configs
mkdir results/
kpt fn run example-configs/ --results-dir results/ --image gcr.io/kpt-functions/validate-rolebinding:results -- subject_name=bob@foo-corp.com
```

### Declarative Run

Functions can be specified declaratively using an annotation on a resource which will
be used as the functionConfig.

#### Run Container Function

Container based functions can be run by specifying them in the functionConfig:

```yaml
apiVersion: example.com/v1beta1
kind: ExampleKind
metadata:
  name: function-input
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gcr.io/a/b:v1
```

And running them on the directory containing the functionConfig yaml with:

```shell script
kpt fn run DIR/
```

By default, container functions cannot access network or volumes.  Functions may enable network
access using the `--network` flag, and specifying that a network is required in the functionConfig.


```yaml
apiVersion: example.com/v1beta1
kind: ExampleKind
metadata:
  name: function-input
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gcr.io/a/b:v1
        network:
          required: true
```

```shell script
kpt fn run DIR/ --network
```

#### Run Starlark Function

Starlark based functions can be run by specifying them in the functionConfig:

```yaml
apiVersion: example.com/v1beta1
kind: ExampleKind
metadata:
  name: function-input
  annotations:
    config.kubernetes.io/function: |
      starlark: {path: a/b/c, name: example-name}
```

And running them on the directory containing the functionConfig yaml with:

```shell script
kpt fn run DIR/ --enable-star
```

[kpt-functions-sdk]: https://github.com/GoogleContainerTools/kpt-functions-sdk