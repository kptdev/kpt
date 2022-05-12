You can develop a KRM function in Go using the kpt function toolkit.

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

## Prerequisites

#### Install kpt

`kpt` is used to get the example Go projects "get-started" and provides an easy way to generate the function specification
object `ResourceList`.
[kpt.dev/installation](https://kpt.dev/installation/)

#### Install Docker

The KRM function can be released as an OCI image or an executable. This guide uses Docker to build
the image. It also shows the executable approach.
[docker installation](https://docs.docker.com/get-docker/)

## Quickstart

### Initialize your project

We start from a "get-started" package which contains a `main.go` file with some scaffolding code.

```shell
export FUNCTION_NAME=<your KRM function name>

# Get the "get-started" package.
kpt pkg get https://github.com/GoogleContainerTools/kpt-functions-sdk.git/go/get-started@master ${FUNCTION_NAME}

cd ${FUNCTION_NAME}

# Initialize the Go module
go mod init
go mod tidy
```

### Write the KRM function logic
 
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
// You can modify the `Items` and add result information to `rl.Result`.
func Run()(rl *fn.ResourceList) (bool, error) {
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

The `fn` library provides a series of KRM level operations for `ResourceList`. 
Basically, the KRM resource `ResourceList.FunctionConfig` and KRM resources `ResourceList.Items` are both converted to 
`KubeObject` objects. You can use `KubeObject` similar as [`unstructured.Unstrucutred`](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured).

To learn more about the `KubeObject`, see https://pkg.go.dev/github.com/GoogleContainerTools/kpt-functions-sdk/go/fn


### Test the KRM function

The "get-started" package contains a `./data` directory. You can use this to test out your functions. 

```shell
# Modify the `data/resources.yaml` with your targeting KRM resource
vim data/resources.yaml

# Define your `ResourceList.FunctionConfig` object. 
vim data/fn-config.yaml

# Convert the KRM resources and FunctionConfig resource to `ResourceList`.
kpt fn source data --fn-config data/fn-config.yaml

# Pass the ResourceList as StdIn to your function
kpt fn source data --fn-config data/fn-config.yaml | go run main.go

# Verify the output `ResourceList`
```

### Build the KRM function as an executable

During iterative development, `--exec` flag can be used to execute the
function binary directly instead of requiring the function to be containerized
first.

After editing your function, you can release it as an executable
```shell
go build main.go -o ${FUNCTION_NAME}
# Change permission mode if needed
chmod -x ${FUNCTION_NAME}
```

#### Run the executable imperatively via kpt
```shell
export PATH=<a path to your KRM resources directory>
export FN_PATH=<a path to your FunctionConfig .yaml file>

kpt fn eval ${PATH} --exec ./${FUNCTION_NAME} --fn-config ${FN_PATH}
```

#### Run the executable declaratively via kpt

You can add the function to `Kptfile` via `--save` flag

```shell
kpt fn eval ${PATH} --save --exec ./${FUNCTION_NAME} --fn-config ${FN_PATH}

# Run the executable in the Kptfile pipeline
kpt fn render --allow-exec ${PATH}
```

Once you have a function binary that works, you can then proceed to
creating the container image.

### Build the KRM function as a Docker image

Generate a Dockerfile for the function image:
```shell
cat > Dockerfile << EOF 
FROM golang:1.17-alpine3.15
ENV CGO_ENABLED=0
WORKDIR /go/src/
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /usr/local/bin/function ./
FROM alpine:3.15
COPY --from=0 /usr/local/bin/function /usr/local/bin/function
ENTRYPOINT ["function"]
EOF
```

Build the image:

```shell
export FN_CONTAINER_REGISTRY=<Your GCR or dockerh host>
$ docker build . -t ${FN_CONTAINER_REGISTRY}
```

To verify the image using the same `./data` resources
```shell
kpt fn eval ${PATH} --image ${FN_CONTAINER_REGISTRY} --fn-config ${FN_PATH}
```
