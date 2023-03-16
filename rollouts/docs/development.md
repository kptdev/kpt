# Development

## Changing Rollouts API

If you change the API resources, in `api/.../*.go`, update the generated code and manifests by running:

```sh
make generate
make manifests
```

Remove current Rollout and RemoteSync objects from your cluster. Then, remove old CRDs and apply the 
new CRDs to your cluster by running:

```sh
make uninstall
make install
```

## Components

Rollouts comprises of several software components:

* [api](../api): Definition of the KRM API supported by the rollouts controllers.
* [config](../config): The generated rollouts manifests.
* [controllers](../controllers): The controllers and related code for the Rollout CRDs. 
* [manifests](../manifests): The generated rollouts manifests in kpt package format.
* [clusterstore](../pkg/clusterstore): Represents a store of kubernetes clusters.
* [packageclustermatcher](../pkg/packageclustermatcher): Matches discovered packages to known clusters.
* [packagediscovery](../pkg/packagediscovery): Discovers config packages to be rolled out.
* [tokenexchange](../pkg/tokenexchange): Exchanges tokens for authenticating GCP service accounts.
* [rolloutsclient](../rolloutsclient): A client for the rollouts API.

## Running Rollouts

See dedicated documentation on running Rollouts:

* [locally](running-locally.md)
* on GKE (getting started guide coming soon)

## Build the Container Images

Build Docker images of Rollout components:

```sh
# Build Images
make docker-build

# Push Images to Docker Registry
make docker-push

# Supported make variables:
# IMG     - image name and tag

# Example:
IMG=gcr.io/<your-project>/rollouts-controller:v0.0.x make docker-push
```

## Running Locally

Follow [running-locally.md](./running-locally.md) to run Rollouts locally.

## Releasing

Follow [releasing.md](./releasing.md) for the current Rollouts release process.

## Makefile Targets

* `make manifests`: generate Rollouts manifests under `config`
* `make generate`: generate code based on Rollouts API definitions (runs k8s code generators)
* `make tidy`: run go mod tidy
* `make fmt`: formats golang sources
* `make vet`: vets golang sources
* `make test`: runs tests
* `make build`: builds manager binary
* `make docker-build`: builds Rollouts Docker images
* `make docker-push`: pushes Rollouts Docker images
* `make install`: install CRDs into the K8s cluster specified in ~/.kube/config
* `make uninstall`: uninstall CRDs from the K8s cluster specified in ~/.kube/config
* `make deploy`: deploy controller to the K8s cluster specified in ~/.kube/config (must provide IMG variable for the controller image)
* `make undeploy`: undeploy controller from the K8s cluster specified in ~/.kube/config
* `make license`: add licenses to source code

## VSCode

[VSCode](https://code.visualstudio.com/) works really well for editing and debugging.
