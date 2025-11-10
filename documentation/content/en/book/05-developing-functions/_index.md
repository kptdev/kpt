---
title: "Chapter 5: Developing Functions"
linkTitle: "Chapter 5: Developing Functions"

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

To simplify development, we provide a framework for developing functions in Go.
We will cover that framework later in this chapter.

### Authoring Executable Configuration

Instead of developing a custom image, you can use an existing function image
containing a language interpreter, and provide your business logic in a KRM
resource. This is referred to as _executable configuration_. We will see two
examples of executable configuration pattern in this chapter.

Although using executable configuration saves some time initially, it can become
an anti-pattern if it grows in complexity. We recommend limiting their use if:

- There is a small amount of logic (< 20 lines)
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
kpt CLI) and functions. This standard was published as the
[KRM Functions Specification](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md#krm-functions-specification)
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
[the KRM function SDK](https://pkg.go.dev/github.com/kptdev/krm-functions-sdk/go/fn). The source code
of the SDK is [available in Github](https://github.com/kptdev/krm-functions-sdk).

[the KRM function SDK](https://pkg.go.dev/github.com/kptdev/krm-functions-sdk/go/fn). The source code
of the SDK is [available in Github](https://github.com/kptdev/krm-functions-sdk).

Go is a general-purpose programming language that is more suitable and robust than Domain Specific Languages (DSL)
for writing functions that manipulate KRM. Go provides:
  - Proper abstractions and language features
  - An extensive ecosystem of tooling (e.g. IDE support)
  - A comprehensive catalog of well-supported libraries
  - Robust community support and detailed documentation

### Prerequisites

- [Install kpt](installation/kpt-cli/)

- [Install Docker](https://docs.docker.com/get-docker/)

- [Golang](https://go.dev/dl/) (at least version 1.24)

### Quickstart

In this quickstart, we will write a function called "set-annotation" that adds an annotation 
`config.kubernetes.io/managed-by=kpt` to all `Deployment` resources.

#### Set up your project

We start from the [get-started](https://github.com/kptdev/krm-functions-sdk/tree/master/go/get-started) package int he KRM Funxtions SDK,
which contains a `main.go` file with some scaffolding code.

Initialize your project.

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
// Copyright 2022-2025 The kpt Authors
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
`KubeObject` objects. You can use `KubeObject` in a similar manner to
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

Learn more about the `KubeObject` from the [go documentation](https://pkg.go.dev/github.com/kptdev/krm-functions-sdk/go/fn#KubeObject).

#### Test the KRM function

The "get-started" package contains a `./testdata` directory. You can use this to test out your functions. 

```shell
# Edit the `testdata/test1/resources.yaml` with your KRM resources. 
# resources.yaml already has a `Deployment` and `Service` as test data. 
vim testdata/test1/resources.yaml

# Convert the KRM resources and FunctionConfig resource to `ResourceList`, and 
# then pipe the ResourceList as StdIn to your function
kpt fn source testdata | go run main.go
```

Verify the KRM function behavior in the StdOutput `ResourceList` by looking for the new annotation on the "nginx-deplyment":

```yaml
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /nginx-deployment
  name: nginx-deployment
  annotations:
    config.kubernetes.io/managed-by: kpt
```

#### Provide function configuration to the KRM function

Let's amend the example to set the "config.kubernetes.io/managed-by" annotation to a value we provide.

Change the implementaiton of the `Run` function above as follows. Change the line:

```shell
kubeObject.SetAnnotation("config.kubernetes.io/managed-by", "kpt")
```

to

```shell
kubeObject.SetAnnotation("config.kubernetes.io/managed-by", r.FnConfigFoo)
```

The annotation value will be set from the value of the `FnConfigFoo` field.

Create the configuration information so that we can concatenate it onto the ResourceList generated by the `kpt fn source` command. This
configuration specifies that the "config.kubernetes.io/managed-by" annotation should be set to a value of "bar".

``` shell
cat > fn-config.yaml  <<\EOF
functionConfig:
  apiVersion: fn.kpt.dev/v1alpha1
  kind: YourFunction
  metadata: # kpt-merge: /test
    name: test
    annotations:
      internal.kpt.dev/upstream-identifier: 'fn.kpt.dev|YourFunction|default|test'
  fnConfigFoo: bar
EOF
```

Run the KRM function
```shell
{kpt fn source testdata; cat fn-config.yaml} | go run main.go
```

Look for the "config.kubernetes.io/managed-by" annotation in the standard output:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /nginx-deployment
  name: nginx-deployment
  annotations:
    config.kubernetes.io/managed-by: bar
```

#### Debug the KRM function in VSCode

Open VSCode in the main directory of the KRM function. Use a ".vscode/launch.json" file to
set the KRM function launch configuration for debugging:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch KRM function",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}",
            "console": "integratedTerminal"
        }
    ]
}
```

You launch your KRM function in VSCode with the "Launch KRM function" configuration. Paste the ResourceList yaml (output of the `kpt fn source testdata`
plus your function configuraiton) into the VSCode terminal and type "ctrl-D" (EOF) so that the KRM function can read the ResourceList from its
standard input. You can now debug the KRM function in the VSCode debugger.

![img](/images/debug-fn-in-vscode.png)


#### Build the KRM function as a Docker image

Build the image

The "get-started" package provides the `Dockerfile` that you can download using:
```shell
wget https://raw.githubusercontent.com/kptdev/krm-functions-sdk/master/go/kfn/commands/embed/Dockerfile
```

```shell
export FN_CONTAINER_REGISTRY=<Your GHCR or docker hub>
export TAG=<Your KRM function tag>
docker build . -t ${FN_CONTAINER_REGISTRY}/${FUNCTION_NAME}:${TAG}
```

To verify the image using the same `./testdata` resources
```shell
kpt fn eval ./testdata/test1/resources.yaml --image ${FN_CONTAINER_REGISTRY}/${FUNCTION_NAME}:${TAG}
```

### Next Steps

- See other [go documentation examples](https://pkg.go.dev/github.com/kptdev/krm-functions-sdk/go/fn/examples) to use KubeObject.
- To contribute to KRM catalog functions, please follow the [contributor guide](https://github.com/kptdev/krm-functions-catalog/blob/master/CONTRIBUTING.md)
