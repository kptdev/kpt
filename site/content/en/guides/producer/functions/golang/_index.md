---
title: "Go Function Libraries"
linkTitle: "Go Function Libraries"
weight: 4
type: docs
description: >
   Writing exec and container functions in Golang.
---

Function developers can write exec and container functions in Golang using the
following libraries:

| Library                                    | Purpose                |
| ------------------------------------------ | ---------------------- |
| [sigs.k8s.io/kustomize/kyaml/fn/framework] | Setup function command |
| [sigs.k8s.io/kustomize/kyaml/yaml]         | Modify resources       |

Develop using the two examples of writing functions in Go or consult the
kyaml libraries' reference below.

## Hello World Go Function

### Create the go module

```sh
go mod init github.com/user/repo
go get sigs.k8s.io/kustomize/kyaml
```

### Create the `main.go`

```go
// main.go
package main

import (
 "os"

 "sigs.k8s.io/kustomize/kyaml/fn/framework"
 "sigs.k8s.io/kustomize/kyaml/yaml"
)

var value string

func main() {
    resourceList := &framework.ResourceList{}
 cmd := framework.Command(resourceList, func() error {
        // cmd.Execute() will parse the ResourceList.functionConfig into
        // cmd.Flags from the ResourceList.functionConfig.data field.
  for i := range resourceList.Items {
            // modify the resources using the kyaml/yaml library:
            // https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml
   if err := resourceList.Items[i].PipeE(yaml.SetAnnotation("value", value)); err != nil {
    return err
   }
  }
  return nil
 })
 cmd.Flags().StringVar(&value, "value", "", "flag value")
 if err := cmd.Execute(); err != nil {
  os.Exit(1)
 }
}
```

### Build and test the function

Build the go binary:

```sh
go build -o my-fn .
```

Test it by running imperatively as an executable function:

```sh
# run the my-fn function against the configuration in PACKAGE_DIR/
kpt fn run PACKAGE_DIR/ --enable-exec --exec-path ./my-fn -- value=foo
```

Or declaratively as an executable function:

```yaml
# PACKAGE_DIR/example.yaml
apiVersion: example.com/v1alpha1
kind: ConfigMap
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: /path/to/my-fn
data:
  value: foo
```

```sh
kpt fn run PACKAGE_DIR/ --enable-exec
```

### Publish the function

Build the function into a container image:

```sh
# optional: generate a Dockerfile to contain the function
go run ./main.go gen ./
```

```sh
# build the function into an image
docker build . -t gcr.io/project/fn-name:tag
# optional: push the image to a container registry
docker push gcr.io/project/fn-name:tag
```

Run the function imperatively as a container function:

```sh
kpt fn run PACKAGE_DIR/ --image gcr.io/project/fn-name:tag -- value=foo
```

Or run declaratively as a container function:

```yaml
# PACKAGE_DIR/example.yaml
apiVersion: example.com/v1alpha1
kind: ConfigMap
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gcr.io/project/fn-name:tag
data:
  value: foo
```

```sh
kpt fn run PACKAGE_DIR/
```

## Advanced Go Function

Functions may alternatively be written using a struct for parsing the
functionConfig rather than flags. The example shown below explicitly
implements what the preceding example implements implicitly.

```go
package main

import (
 "os"

 "sigs.k8s.io/kustomize/kyaml/fn/framework"
 "sigs.k8s.io/kustomize/kyaml/yaml"
)

func main() {
 type Data struct {
        Value string `yaml:"value,omitempty"`
 }
 type Example struct {
        // Data contains the function configuration (e.g. client-side CRD).
        // Using "data" as the field name to contain key-value pairs enables
        // the function to be invoked imperatively via
        // `kpt fn run DIR/ --image img:tag -- key=value` and the
        // key=value arguments will be parsed into the functionConfig.data
        // field.
        // If the function does not need to be invoked imperatively, other
        // field names may be used.
        Data Data `yaml:"data,omitempty"`
 }
 functionConfig := &Example{}
    resourceList := &framework.ResourceList{FunctionConfig: functionConfig}

 cmd := framework.Command(resourceList, func() error {
  for i := range resourceList.Items {
            // use the kyaml libraries to modify each resource by applying
            // transformations
   err := resourceList.Items[i].PipeE(
                yaml.SetAnnotation("value", functionConfig.Data.Value),
            )
            if err != nil {
    return nil, err
   }
  }
  return items, nil
 })

 if err := cmd.Execute(); err != nil {
  os.Exit(1)
 }
}
```

Note: functionConfig need not read from the `data` field if it is not going to
be run imperatively with
`kpt fn run DIR/ --image gcr.io/some/image -- foo=bar` or
`kpt fn run DIR/ --exec-path /some/bin --enable-exec -- foo=bar`. This is more
appropriate for functions implementing abstractions (e.g. client-side CRD
equivalents).

```go
...
 type NestedValue struct {
  Value string `yaml:"value,omitempty"`
 }
 type Spec struct {
        NestedValue string `yaml:"nestedValue,omitempty"`
        MapValues map[string]string  `yaml:"mapValues,omitempty"`
        ListItems []string  `yaml:"listItems,omitempty"`
 }
 type Example struct {
        Spec Spec `yaml:"spec,omitempty"`
 }
 functionConfig := &Example{}
...
```

```yaml
# PACKAGE_DIR/example.yaml
apiVersion: example.com/v1alpha1
kind: Example
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: /path/to/my-fn
spec:
  nestedValue:
    value: something
  mapValues:
    key: value
  listItems:
    - a
    - b
```

## kyaml Libraries Reference

Functions written in go should use the [sigs.k8s.io/kustomize/kyaml] libraries
for modifying resource configuration.

The [sigs.k8s.io/kustomize/kyaml/yaml] library offers utilities for reading
and modifying yaml configuration, while retaining comments and structure.

To use the kyaml/yaml library, become familiar with:

- The `*yaml.RNode` type, which represents a configuration object or field
  - [RNode link]
- The `Pipe` and `PipeE` functions, which apply a series of pipelined
  operations to the `*RNode`.
  - [Pipe link]

### Workflow

To modify a `*yaml.RNode` call PipeE() on the `*RNode`, passing in the
operations to be performed.

```go
// Set the spec.replicas field to 3 if it exists
var node *yaml.RNode
...
err := node.PipeE(yaml.Lookup("spec", "replicas"), yaml.FieldSetter{StringValue: "3"})
```

```go
// Set the spec.replicas field to 3, creating it if it doesn't exist
var node *yaml.RNode
...
// pass in the type of the node to create if it doesn't exist
// (e.g. Sequence, Map, Scalar)
err := node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "spec", "replicas"), yaml.FieldSetter{StringValue: "3"})
```

To read a value from a \*yaml.RNode call Pipe() on the RNode, passing in the
operations to lookup a field.

```go
// Read the spec.replicas field
var node *yaml.RNode
...
replicas, err := node.Pipe(yaml.Lookup("spec", "replicas"))
```

{{% pageinfo color="info" %}}
Operations are any types implementing the `yaml.Filter` interface, so it is
simple to define custom operations and provide them to `Pipe`, combining them
with the built-in operations.
{{% /pageinfo %}}

### Visiting Fields and Elements

Maps (i.e. Objects) and Sequences (i.e. Lists) support functions for visiting
their fields and elements.

```go
// Visit each of the elements in a Sequence (i.e. a List)
err := node.VisitElements(func(elem *yaml.RNode) error {
    // do something with each element in the list
    return nil
})
```

```go
// Visit each of the fields in a Map (i.e. an Object)
err := node.VisitFields(func(n *yaml.MapNode) error {
    // do something with each field in the map / object
    return nil
})
```

### Validation Results

Go functions can implement high fidelity validation results by setting a
`framework.Result` on the `ResourceList`.

If run using `kpt fn run --results-dir SOME_DIR/`, the result will be written
to a file in the specified directory.

If the result is returned and contains an item with severity of
`framework.Error`, the function will exit non-0. Otherwise it will exit 0.

```go
cmd := framework.Command(resourceList, func() error {
    ...
    if ... {
        // return validation results to be written under the results dir
     resourceList.Result = framework.Result{...}

        // return the results as an error if desireds
        return resourceList.Result
    }
    ...
})

```

## Next Steps

- Learn how to [run functions].
- Consult the [fn command reference].
- Find out how to structure a pipeline of functions from the
  [functions concepts] page.

[sigs.k8s.io/kustomize/kyaml/fn/framework]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/fn/framework/
[sigs.k8s.io/kustomize/kyaml/yaml]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml/
[sigs.k8s.io/kustomize/kyaml]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/
[RNode link]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml/#RNode
[Pipe link]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml/#RNode.Pipe
[run functions]: ../../../consumer/function/
[fn command reference]: ../../../../reference/fn/
[functions concepts]: ../../../../concepts/functions/
