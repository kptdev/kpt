You can use the `kyaml` library to develop functions in Go. Doing so provides
the following features:

- **General-purpose language:** Compared to Domain Specific Languages (DSL), Go
  is a general-purpose programming language that provides:
  - Proper abstractions and language features
  - An extensive ecosystem of tooling (e.g. IDE support)
  - A comprehensive catalog of well-supported libraries
  - Robust community support and detailed documentation
- **YAML-centric**: As opposed to other frameworks discussed in this chapter,
  the `kyaml` library exposes the YAML Abstract Syntax Tree (AST) to the user.
  This enables you to control every aspect of the YAML file including
  manipulating comments; however, it comes at the cost of complexity compared to
  representing resources as idiomatic data structures.

## Quickstart

### Create the go module

```shell
$ go mod init github.com/user/repo; go get sigs.k8s.io/kustomize/kyaml@v0.12.0
```

### Create the `main.go`

This is a simple function that adds the annotation `myannotation` with the
provided value:

```go
// main.go
package main

import (
	"os"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func main() {
	// create a struct matching the structure of ResourceList.FunctionConfig to hold its data
	var config struct {
		Data map[string]string `yaml:"data"`
	}
	fn := func(items []*yaml.RNode) ([]*yaml.RNode, error) {
		for i := range items {
			// set the annotation on each resource item
			err := items[i].PipeE(yaml.SetAnnotation("myannotation", config.Data["myannotation"]))
			if err != nil {
				return nil, err
			}
		}
		return items, nil
	}
	p := framework.SimpleProcessor{Filter: kio.FilterFunc(fn), Config: &config}
	cmd := command.Build(p, command.StandaloneDisabled, false)
  // Adds a "gen" subcommand to create a Dockerfile for building the function into a container image.
  command.AddGenerateDockerfile(cmd)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

### Build and test the function

Build the go binary:

```shell
$ go mod tidy; go build -o my-fn .
```

Fetch the wordpress package:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress@v0.8
```

Test it by running the function imperatively:

```shell
$ kpt fn eval wordpress --exec ./my-fn -- myannotation=foo
```

During iterative development, `--exec` flag can be used to execute the
function binary directly instead of requiring the function to be containerized
first. Once you have a function binary that works, you can then proceed to
creating the container image.

### Publish the function

Generate a Dockerfile for the function image:

```shell
$ go run ./main.go gen ./
```

Build the image:

```shell
$ docker build . -t gcr.io/project/fn-name:tag
```

Optionally, push the image to a container registry:

```shell
$ docker push gcr.io/project/fn-name:tag
```

Run the function imperatively as a container function:

```shell
$ kpt fn eval wordpress -i gcr.io/project/fn-name:tag -- myannotation=foo
```

## Next Steps

- Read the package documentation:

| Package                                    | Purpose               |
| ------------------------------------------ | --------------------- |
| [sigs.k8s.io/kustomize/kyaml/fn/framework] | Functions Framework   |
| [sigs.k8s.io/kustomize/kyaml/yaml]         | Modify YAML resources |

- Take a look at the source code for [functions in the catalog] to better
  understand how to develop functions in Go

[sigs.k8s.io/kustomize/kyaml/fn/framework]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml@v0.11.1/fn/framework#pkg-index
[sigs.k8s.io/kustomize/kyaml/yaml]: https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml@v0.11.1/yaml
[functions in the catalog]: https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/master/functions/go
