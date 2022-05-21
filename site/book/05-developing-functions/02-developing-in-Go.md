You can develop a KRM function in Go using [the kpt function SDK].

- **General-purpose language:** Compared to Domain Specific Languages (DSL), Go
  is a general-purpose programming language that provides:
  - Proper abstractions and language features
  - An extensive ecosystem of tooling (e.g. IDE support)
  - A comprehensive catalog of well-supported libraries
  - Robust community support and detailed documentation

## Prerequisites

- [Install kpt](https://kpt.dev/installation/)

- [Install Docker](https://docs.docker.com/get-docker/)

## Quickstart

In this quickstart, we will write a function that adds an annotation 
`config.kubernetes.io/managed-by=kpt` to all `Deployment` resources.

### Initialize your project

We start from a "get-started" package which contains a `main.go` file with some scaffolding code.

```shell
# Set your KRM function name.
export FUNCTION_NAME=set-annotation

# Get the "get-started" package.
kpt pkg get https://github.com/GoogleContainerTools/kpt-functions-sdk.git/go/get-started@master ${FUNCTION_NAME}

cd ${FUNCTION_NAME}

# Initialize the Go module
go mod init
go mod tidy
```

### Write the KRM function logic
 
Take a look at the `main.go` (as below) and complete the `Run` function.

```go
// main.go
package main

import (
  "fmt"
  "os"

  "github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
)

// EDIT THIS FUNCTION!
// This is the main logic. rl is the input `ResourceList` which has the `FunctionConfig` and `Items` fields.
// You can modify the `Items` and add result information to `rl.Results`.
func Run(rl *fn.ResourceList) (bool, error) {
  // Your code
}

func main() {
  // CUSTOMIZE IF NEEDED
  // `AsMain` accepts a `ResourceListProcessor` interface.
  // You can explore other `ResourceListProcessor` structs in the SDK or define your own.
  if err := fn.AsMain(fn.ResourceListProcessorFunc(Run)); err != nil {
    os.Exit(1)
  }
}
```

The [`fn`] library provides a series of KRM level operations for [`ResourceList`]. 
Basically, the KRM resource `ResourceList.FunctionConfig` and KRM resources `ResourceList.Items` are both converted to 
`KubeObject` objects. You can use `KubeObject` similar as [`unstructured.Unstrucutred`].

The set-annotation function (see below) iterates the `ResourceList.Items`, finds out the `Deployment` resources and
adds the annotation. After the iteration, it adds some user message to the `ResourceList.Results`

```go
func Run(rl *fn.ResourceList) (bool, error) {
    for _, kubeObject := range rl.Items {
        if kubeObject.IsGVK("apps/v1", "Deployment") {
            kubeObject.SetAnnotation("config.kubernetes.io/managed-by", "kpt")
        }
    }
    // This result message will be displayed in the function evaluation time. 
    rl.Results = append(rl.Results, fn.GeneralResult("Add config.kubernetes.io/managed-by=kpt to all `Deployment` resources", fn.Info))
    return true, nil
}
```

Learn more about the `KubeObject` from the [go doc](https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn).


### Test the KRM function

The "get-started" package contains a `./data` directory. You can use this to test out your functions. 

```shell
# Edit the `data/resources.yaml` with your KRM resources. 
# resources.yaml already has a `Deployment` and `Service` as test data. 
vim data/resources.yaml

# Convert the KRM resources and FunctionConfig resource to `ResourceList`, and 
# then pipe the ResourceList as StdIn to your function
kpt fn source data | go run main.go

# Verify the KRM function behavior in the StdOutput `ResourceList`
```

### Build the KRM function as a Docker image

Build the image

The "get-started" package provides the `Dockerfile`.

```shell
export FN_CONTAINER_REGISTRY=<Your GCR or docker hub>
export TAG=<Your KRM function tag>
docker build . -t ${FN_CONTAINER_REGISTRY}/${FUNCTION_NAME}:${TAG}
```

To verify the image using the same `./data` resources
```shell
kpt fn eval ./data --image ${FN_CONTAINER_REGISTRY}/${FUNCTION_NAME}:${TAG}
```

## Next Steps

- See other [go doc examples] to use KubeObject.
- To contribute to KRM catalog functions, please follow the [contributor guide](https://github.com/GoogleContainerTools/kpt-functions-catalog/blob/master/CONTRIBUTING.md)

[the kpt function SDK]: https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn
[go doc examples]: https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn/examples
[`fn`]: https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn
[`ResourceList`]: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md
[`unstructured.Unstrucutred`]: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured
