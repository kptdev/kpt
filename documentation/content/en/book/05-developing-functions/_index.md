---
title: "Chapter 5: Developing-functions"
linkTitle: "Chapter 5: Developing-functions"
description: |
    [Chapter 2](../02-concepts/#functions) provided a high-level conceptual explanation of functions. We discussed how
    this architecture enables us to develop functions in different languages, frameworks and runtimes. In this chapter,
    we are going to look at different approaches to developing functions.

toc: true
menu:
  main:
    parent: "Book"
    weight: 50
---

 > Before you start developing your custom function, check out the
[Functions Catalog](https://catalog.kpt.dev/function-catalog ":target=_self") in case there is
an existing function that meets your needs. This is an ever-growing catalog of
functions that we consider to be generally useful to many users. If your use
case fits that description, please
[open a feature request](https://github.com/kptdev/kpt/issues/new?assignees=&labels=enhancement&template=feature_request.md&title=) for adding a function to the catalog.

## Approaches

### Creating Custom Images

With this approach you create a custom container image which can execute
programs in an arbitrary language or encapsulate existing tools as long as it
satisfies the KRM Functions Specification we will see later in this chapter.

To simplify development, we provide frameworks for developing functions in Go
and Typescript. We will cover these later in this chapter.

### Authoring Executable Configuration

Instead of developing a custom image, you can use an existing function image
containing a language interpreter, and provide your business logic in a KRM
resource. This is referred to as _executable configuration_. We will see two
examples of executable configuration pattern in this chapter.

Although using executable configuration saves some time initially, it can become
an anti-pattern if it grows in complexity. We recommend limiting their use to:

- Small amount of logic (< 20 lines)
- You do not forsee this logic growing in complexity in the future

Otherwise, you are better off developing functions in a general-purpose language
where you can take advantage of proper abstractions and language features,
better testing, rich IDE experience, and existing libraries.

## Function Properties

As you think about how to formulate your function, keep in mind the following
desired properties:

### Deterministic

Executing a function with the same input should produce the same output. For
example, a function that annotates a resource with the current timestamp is not
deterministic.

Note that input to the function includes both `input items` and the
`functionConfig`:

![img](/images/func.svg)

### Idempotent

Executing a function a second time should not produce any change. For example, a
function that increments value of the `replicas` field by 1 is not idempotent.
Instead, the function could take the desired value of the `replicas` field as
input.

This property enables in-place edits to work and is analogous to the
level-driven reconciliation model of the Kubernetes system.

### Hermetic and Unprivileged

If possible, try to formulate your function to be hermetic. We discussed this in
detail in [chapter 4](../04-using-functions#privileged-execution).

## Functions Specification

In order to enable functions to be developed in different toolchains and
languages and be interoperable and backwards compatible, the kpt project created
a standard for the inter-process communication between the orchestrator (i.e.
kpt CLI) and functions. This standard was published as [KRM Functions Specification](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md#krm-functions-specification)
and donated to the CNCF as part of the Kubernetes SIG-CLI.

Understanding this specification enables you to have a deeper understanding of
how things work under the hood. It also enables to create your own toolchain for
function development if you so desire.

As an example, you can see the `ResourceList` containing resources in the
`wordpress` package:

```shell
kpt fn source wordpress | less
```

## Developing in Go

You can develop a KRM function in Go using
[the kpt function SDK](https://pkg.go.dev/github.com/kptdev/krm-functions-sdk/go/fn).

- **General-purpose language:** Compared to Domain Specific Languages (DSL), Go
  is a general-purpose programming language that provides:
  - Proper abstractions and language features
  - An extensive ecosystem of tooling (e.g. IDE support)
  - A comprehensive catalog of well-supported libraries
  - Robust community support and detailed documentation

### Prerequisites

- [Install kpt](installation)

- [Install Docker](https://docs.docker.com/get-docker/)

- [Golang](https://go.dev/dl/) (at least version 1.19)

### Quickstart

In this quickstart, we will write a function that adds an annotation 
`config.kubernetes.io/managed-by=kpt` to all `Deployment` resources.

#### Initialize your project

We start from a "get-started" package which contains a `main.go` file with some scaffolding code.

```shell
# Set your KRM function name.
export FUNCTION_NAME=set-annotation

# Get the "get-started" package.
kpt pkg get https://github.com/kptdev/krm-functions-sdk.git/go/get-started@master ${FUNCTION_NAME}

cd ${FUNCTION_NAME}

# Initialize the Go module
go mod init
go mod tidy
```

#### Write the KRM function logic
 
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

	"github.com/kptdev/krm-functions-sdk/go/fn"
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

The [`fn`](https://pkg.go.dev/github.com/kptdev/krm-functions-sdk/go/fn) library provides a series of KRM level
operations for
[`ResourceList`](https://pkg.go.dev/github.com/kptdev/krm-functions-sdk/go/fn#ResourceList). 
Basically, the KRM resource `ResourceList.FunctionConfig` and KRM resources `ResourceList.Items` are both converted to 
`KubeObject` objects. You can use `KubeObject` similar as
[`unstructured.Unstructured`](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured).

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

Learn more about the `KubeObject` from the [go doc](https://pkg.go.dev/github.com/kptdev/krm-functions-sdk/go/fn#KubeObject).


#### Test the KRM function

The "get-started" package contains a `./testdata` directory. You can use this to test out your functions. 

```shell
# Edit the `testdata/test1/resources.yaml` with your KRM resources. 
# resources.yaml already has a `Deployment` and `Service` as test data. 
vim testdata/test1/resources.yaml

# Convert the KRM resources and FunctionConfig resource to `ResourceList`, and 
# then pipe the ResourceList as StdIn to your function
kpt fn source testdata | go run main.go

# Verify the KRM function behavior in the StdOutput `ResourceList`
```

#### Build the KRM function as a Docker image

Build the image

The "get-started" package provides the `Dockerfile` that you can download using:
```shell
wget https://raw.githubusercontent.com/kptdev/krm-functions-sdk/master/go/kfn/commands/embed/Dockerfile
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

### Next Steps

- See other [go doc examples](https://pkg.go.dev/github.com/kptdev/krm-functions-sdk/go/fn/examples) to use KubeObject.
- To contribute to KRM catalog functions, please follow the [contributor guide](https://github.com/kptdev/krm-functions-catalog/blob/master/CONTRIBUTING.md)
