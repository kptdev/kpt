# Running Porch Locally

## Prerequisites

To run Porch locally, you will need:

* Linux machine (technically it is possible to run Porch locally on a Mac but
  due to differences in Docker between Linux and Mac, the Porch scripts are
  confirmed to work on Linux)
* [go 1.17](https://go.dev/dl/) or newer
* [docker](https://docs.docker.com/get-docker/)
* [git](https://git-scm.com/)
* `make`

## Getting Started

Clone this repository into `${GOPATH}/src/github.com/GoogleContainerTools/kpt`.

```sh
git clone https://github.com/GoogleContainerTools/kpt.git "${GOPATH}/src/github.com/GoogleContainerTools/kpt"
```

Download dependencies:

```sh
make tidy
```

## Running Porch

Porch consists of:
* k8s extension apiserver [porch](../porch/apiserver)
* kpt function evaluator [func](../porch/func)
* k8s [controllers](../porch/controllers)

In addition, to run Porch locally, we need to run the main k8s apiserver and its backing storage, etcd.

To build and run Porch locally in one command, run:

```sh
# Go into the porch directory:
cd "${GOPATH}/src/github.com/GoogleContainerTools/kpt/porch"

# Start Porch in one command:
make
```

This will:

* create Docker network named `porch`
* build and start `etcd` Docker container
* build and start main k8s apiserver Docker container
* build and start the kpt function evaluator microservice [func](../porch/func) Docker container
* build Porch binary and run it locally
* configure Porch as the extension apiserver

**Note:** this command does not build and start the Poprch k8s controllers. Those
are not required for basic package orchestration but are required for deploying packages.

You can also run the commands individually which can be useful when developing,
in particular building and runnng Porch extension apiserver.

```sh
# Create Porch network
make network

# Build and start etcd container
make start-etcd

# Build and start main apiserver container
make start-kube-apiserver

# Build and start kpt function evaluator microservice Docker container
make start-function-runner

# Build and start Porch on your local machine.
make run-local
```

Porch will run directly on your local machine and API requests will be forwarded to it from the
main apiserver. Configure `kubectl` context to interact with the main k8s apiserver running as
Docker container:

```sh
export KUBECONFIG=${PWD}/hack/local/kubeconfig

# Confirm Porch is running
kubectl api-resources | grep porch

repositories                  config.porch.kpt.dev/v1alpha1          true         Repository
functions                     porch.kpt.dev/v1alpha1                 true         Function
packagerevisionresources      porch.kpt.dev/v1alpha1                 true         PackageRevisionResources
packagerevisions              porch.kpt.dev/v1alpha1                 true         PackageRevision
```

## Restarting Porch

If you make code changes, an expedient way to rebuild and restart porch is:

* Stop Porch running in the shell session (Ctrl+C)
* Run `make run-local` again to rebuild and restart Porch

## Stopping Porch

To stop Porch and all associated Docker containers, including the Docker network, run:

```sh
make stop
```
