---
title: "Go Function SDK"
linkTitle: "Go Function SDK"
weight: 4
type: docs
description: >
   Writing exec and container functions in Golang.
---

Writing exec and container functions in Golang.

### Hello World Go Function

#### Create the go module

```sh
go mod init github.com/user/repo
```

#### Create the `main.go`

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
	cmd := framework.Command(nil, func(items []*yaml.RNode) ([]*yaml.RNode, error) {
        // framework has parse the ResourceList.functionConfig input into the
        // cmd flags(from the ResourceList.functionConfig.data field).
		for i := range items {
            // modify the resources using the kyaml/yaml library:
            // https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml
			if err := items[i].PipeE(yaml.SetAnnotation("value", value)); err != nil {
				return nil, err
			}
		}
		return items, nil
	})
	cmd.Flags().StringVar(&value, "value", "", "flag value")
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

Note: resources should be modified using the [kyaml/yaml](#kyaml) library.

### Build and test the function

Build the go binary and test it by running it as an executable function.

```sh
go build -o my-fn .
```

```sh
# run the my-fn function against the configuration in PACKAGE_DIR/
kpt fn run PACKAGE_DIR/ --enable-exec --exec-path ./my-fn -- value=foo
```

### Publish the function

Build the function into a container image.

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

Run the function as a container

```sh
kpt fn run PACKAGE_DIR/ --image gcr.io/project/fn-name:tag -- value=foo
```

### Declarative function configuration

#### Run the function declaratively

Run as a container function:

```yaml
# PACKAGE_DIR/example.yaml
apiVersion: example.com/v1alpha1
kind: Example
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gcr.io/project/fn-name:tag
data:
  value: a
```

```sh
kpt fn run PACKAGE_DIR/
```

Or as an exec function:

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
data:
  value: a
```

```sh
kpt fn run PACKAGE_DIR/ --enable-exec
```

#### Implement the function using declarative input

Functions may alternatively be written using a struct for parsing the functionConfig rather than
flags.  The example shown below explicitly implements what the preceding example implements
implicitly.

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
        Data Data `yaml:"data,omitempty"`
	}
	functionConfig := &Example{}

	cmd := framework.Command(functionConfig, func(items []*yaml.RNode) ([]*yaml.RNode, error) {
        // framework has parsed the input ResourceList.functionConfig into the functionConfig
        // variable
		for i := range items {
			if err := items[i].PipeE(yaml.SetAnnotation("value", functionConfig.Data.Value)); err != nil {
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

Note: functionConfig need not read from the `data` field if it is not going to be run
imperatively with `kpt fn run DIR/ --image gcr.io/some/image -- foo=bar` or 
`kpt fn run DIR/ --exec-path /some/bin --enable-exec -- foo=bar`.  This is more appropriate
for functions implementing abstractions (e.g. client-side CRD equivalents).

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
### kyaml

Functions written in go should use the [sigs.k8s.io/kustomize/kyaml] libraries for modifying
resource configuration.

The [sigs.k8s.io/kustomize/kyaml/yaml] library offers utilities for reading and modifying
yaml configuration, while retaining comments and structure.

To use the kyaml/yaml library, become familiar with:
 
- The `*yaml.RNode` type, which represents a configuration object or field
  - [link](https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml?tab=doc#RNode)
- The `Pipe` and `PipeE` functions, which apply a series of pipelined operations to the `*RNode`.
  - [link](https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml?tab=doc#RNode.Pipe)


#### Workflow

To modify a *yaml.RNode call PipeE() on the *RNode, passing in the operations to be performed.

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
// pass in the type of the node to create if it doesn't exist (e.g. Sequence, Map, Scalar)
err := node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "spec", "replicas"), yaml.FieldSetter{StringValue: "3"})
```

To read a value from a *yaml.RNode call Pipe() on the RNode, passing in the operations to
lookup a field.

```go
// Read the spec.replicas field
var node *yaml.RNode
...
replicas, err := node.Pipe(yaml.Lookup("spec", "replicas"))
```

{{% pageinfo color="info" %}}
Operations are any types implementing the `yaml.Filter` interface, so it is simple to
define custom operations and provide them to `Pipe`, combining them with the built-in operations.
{{% /pageinfo %}}


#### Visiting Fields and Elements

Maps (i.e. Objects) and Sequences (i.e. Lists) support functions for visiting their fields and
elements.

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

### Validation

Go functions can implement high fidelity validation results by returning a `framework.Result` as
an error.  If run using `kpt fn run --results-dir SOME_DIR/`, the results will be written to a file
in the specified directory.

If the result contains an item with severity of `framework.Error`, the function will exit non-0.
Otherwise it will exit 0.


```go
cmd := framework.Command(functionConfig, func(items []*yaml.RNode) ([]*yaml.RNode, error) {
    ...
    if ... {
        // return validation results to be written under the results dir
        return items, framework.Result{...}
    }
    ...
})

```

[sigs.k8s.io/kustomize/kyaml]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml?tab=doc
[sigs.k8s.io/kustomize/kyaml/yaml]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml?tab=doc
