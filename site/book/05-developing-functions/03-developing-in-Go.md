You can use the `kyaml` library to development functions in Go. It provides the following
features:

- **General-purpose language:** Compared to Domain Specific Languages (DSL), Go is a general-purpose
  programming language that provides:
  - Proper abstractions and language features
  - A extensive ecosystem of tooling (e.g. IDE support)
  - A comprehensive catalog of well-supported libraries
  - Robust community support and detailed documentation
- **YAML-centric**: As opposed to other frameworks discussed in this chapter, the `kyaml`
  library exposes the YAML Abstract Syntax Tree (AST) to the user. This enables you to
  control every aspect of the YAML file including manipulating comments, but comes at the cost
  of complexity compared to representing resources as idiomatic data structures.

## Quickstart

### Create the go module

```shell
$ go mod init github.com/user/repo
$ go get sigs.k8s.io/kustomize/kyaml@v0.10.6
```

### Create the `main.go`

This is a simple function that adds the annotation `myannotation` with the provided value:

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
			if err := resourceList.Items[i].PipeE(yaml.SetAnnotation("myannotation", value)); err != nil {
				return err
			}
		}
		return nil
	})
	cmd.Flags().StringVar(&value, "value", "", "annotation value")
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

### Build and test the function

Build the go binary:

```shell
$ go build -o my-fn .
```

Test it by running imperatively as an executable function:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress@v0.3
# run the my-fn function against the configuration in wordpress
$ kpt fn eval wordpress --exec-path ./my-fn -- value=foo
```

### Publish the function

Build the function into a container image:

```shell
# optional: generate a Dockerfile to contain the function
$ go run ./main.go gen ./
```

```shell
# build the function into an image
$ docker build . -t gcr.io/project/fn-name:tag
# optional: push the image to a container registry
$ docker push gcr.io/project/fn-name:tag
```

Run the function imperatively as a container function:

```shell
$ kpt fn run PACKAGE_DIR/ --image gcr.io/project/fn-name:tag -- value=foo
```

## Next Steps

- Read the package documentation:

| Package                                    | Purpose               |
| ------------------------------------------ | --------------------- |
| [sigs.k8s.io/kustomize/kyaml/fn/framework] | Functions Framework   |
| [sigs.k8s.io/kustomize/kyaml/yaml]         | Modify YAML resources |

- Take a look at [catalog functions] to better understand how to use the develop functions in Go

[sigs.k8s.io/kustomize/kyaml/fn/framework]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml@v0.10.16/fn/framework#pkg-index
[sigs.k8s.io/kustomize/kyaml/yaml]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml@v0.10.16/yaml
[sigs.k8s.io/kustomize/kyaml]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/
[rnode link]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml/#RNode
[pipe link]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml/#RNode.Pipe
[catalog functions]: https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/master/functions/go
