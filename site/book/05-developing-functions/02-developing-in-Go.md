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

- [Golang](https://go.dev/dl/) (at least version 1.19)

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
// Copyright 2022 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"os"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
)

var _ fn.Runner = &YourFunction{}

// TODO: Change to your functionConfig "Kind" name.
type YourFunction struct {
	FnConfigBool bool
	FnConfigInt  int
	FnConfigFoo  string
}

// Run is the main function logic.
// `items` is parsed from the STDIN "ResourceList.Items".
// `functionConfig` is from the STDIN "ResourceList.FunctionConfig". The value has been assigned to the r attributes
// `results` is the "ResourceList.Results" that you can write result info to.
func (r *YourFunction) Run(ctx *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects, results *fn.Results) bool {
	// TODO: Write your code.
	return true
}

func main() {
	runner := fn.WithContext(context.Background(), &YourFunction{})
	if err := fn.AsMain(runner); err != nil {
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
func (r *YourFunction) Run(ctx *fn.Context, functionConfig *fn.KubeObject, items fn.KubeObjects, results *fn.Results) bool {
	for _, kubeObject := range items {
		if kubeObject.IsGVK("apps", "v1", "Deployment") {
			kubeObject.SetAnnotation("config.kubernetes.io/managed-by", "kpt")
		}
	}
	// This result message will be displayed in the function evaluation time.
	*results = append(*results, fn.GeneralResult("Add config.kubernetes.io/managed-by=kpt to all `Deployment` resources", fn.Info))
	return true
}
```

Learn more about the `KubeObject` from the [go doc](https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn).


### Test the KRM function

The "get-started" package contains a `./testdata` directory. You can use this to test out your functions. 

```shell
# Edit the `testdata/resources.yaml` with your KRM resources. 
# resources.yaml already has a `Deployment` and `Service` as test data. 
vim data/resources.yaml

# Convert the KRM resources and FunctionConfig resource to `ResourceList`, and 
# then pipe the ResourceList as StdIn to your function
kpt fn source testdata | go run main.go

# Verify the KRM function behavior in the StdOutput `ResourceList`
```

### Build the KRM function as a Docker image

Build the image

The "get-started" package provides the `Dockerfile` that you can download using:
```shell
wget https://raw.githubusercontent.com/GoogleContainerTools/kpt-functions-sdk/master/go/kfn/commands/embed/Dockerfile
```

```shell
export FN_CONTAINER_REGISTRY=<Your GCR or docker hub>
export TAG=<Your KRM function tag>
docker build . -t ${FN_CONTAINER_REGISTRY}/${FUNCTION_NAME}:${TAG}
```

To verify the image using the same `./testdata` resources
```shell
kpt fn eval ./testdata/test1/resources.yaml --image ${FN_CONTAINER_REGISTRY}/${FUNCTION_NAME}:${TAG}
```

## Next Steps

- See other [go doc examples] to use KubeObject.
- To contribute to KRM catalog functions, please follow the [contributor guide](https://github.com/GoogleContainerTools/kpt-functions-catalog/blob/master/CONTRIBUTING.md)

[the kpt function SDK]: https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn
[go doc examples]: https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn/examples
[`fn`]: https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn
[`ResourceList`]: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md
[`unstructured.Unstructured`]: https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured
