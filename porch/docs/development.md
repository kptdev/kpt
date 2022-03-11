# Development

## Changing Porch API

If you change the API resources, in `api/porch/.../*.go`, update the generated code by running:

```sh
make generate
```

## Components

Porch comprises of several software components:

* [api](../api): Definition of the KRM API supported by the Porch extension apiserver
* [apiserver](../apiserver): The Porch apiserver implementation, REST handlers, Porch `main` function
* [engine](../engine): Core logic of Package Orchestration - operations on package contents
* [func](../func): KRM function evaluator microservice; exposes gRPC API
* [repository](../repository): Integration with Git and OCI package repositories, and package caching
* [controllers](../controllers): `Repository` CRD. No controller;
  Porch apiserver watches these resources for changes as repositories are (un-)registered.
* [remoterootsync](../controllers/remoterootsync): CRD and controller for deploying packages
* [test](../test): Test Git Server for Porch e2e testing

## Running Porch

See dedicated documentation on running Porch:

* [locally](running-locally.md)
* [on GKE](running-on-gke.md)

## Build the Container Images

Build Docker images of Porch components:

```sh
# Build Images
make build-images

# Push Images to Docker Registry
make push-images

# Supported make variables:
# IMAGE_TAG      - image tag, i.e. 'latest' (defaults to 'latest')
# GCP_PROJECT_ID - GCP project hosting gcr.io repository (will translate to gcr.io/${GCP_PROJECT_ID})
# IMAGE_REPO     - overwrites the default image repository

# Example:
IMAGE_TAG=$(git rev-parse --short HEAD) make push-images
```

## Running Locally

Follow [running-locally.md](./running-locally-md) to run Porch locally.

## Debugging

To debug Porch, run Porch locally [running-locally.md](./running-locally-md), exit porch server running in the shell,
and launch Porch under the debugger. VSCode debug session is preconfigured in [launch.json](../.vscode/launch.json).

Update the launch arguments to your needs.

## Code Pointers

Some useful code pointers:

* Porch REST API handlers in [registry/porch](../apiserver/pkg/registry/porch), for example
  [packagerevision.go](../apiserver/pkg/registry/porch/packagerevision.go)
* Background task handling cache updates in [background.go](../apiserver/pkg/registry/porch/background.go)
* Git repository integration in [repository/pkg/git](../repository/pkg/git)
* OCI repository integration in [repository/pkg/oci](../repository/pkg/oci)
* CaD Engine in [engine](../engine/pkg/engine)
* e2e tests in [e2e](../apiserver/pkg/e2e). See below more on testing.

## Running Tests

All tests can be run using `make test`. Individual tests can be run using `go test`.
End-to-End tests assume that Porch instance is running and `KUBECONFIG` is configured
with the instance. The tests will automatically detect whether they are running against
Porch running on local machien or k8s cluster and will start Git server appropriately,
then run test suite against the Porch instance.

## Makefile Targets

* `make generate`: generate code based on Porch API definitions (runs k8s code generators)
* `make tidy`: tidies all Porch modules
* `make fmt`: formats golang sources
* `make build-images`: builds Porch Docker images
* `make push-images`: builds and pushes Porch Docker images
* `make deployment-config`: customizes configuration which installs Porch
   in k8s cluster with correct image names, annotations, service accounts.
   The deployment-ready configuration is copied into `./.build/deploy`
* `make deploy`: deploys Porch in the k8s cluster configured with current kubectl context
* `make push-and-deploy`: builds, pushes Porch Docker images, creates deployment configuration, and deploys Porch
* `make` or `make all`: builds and runs Porch [locally](./running-locally.md)
* `make test`: runs tests