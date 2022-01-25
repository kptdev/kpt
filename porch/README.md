# Package Orchestration apiserver

Created from https://github.com/kubernetes/sample-apiserver


## Getting Started

Clone this repository into `${GOPATH}/src/github.com/GoogleContainerTools/kpt`.

```sh
git clone https://github.com/GoogleContainerTools/kpt.git "${GOPATH}/src/github.com/GoogleContainerTools/kpt"
cd "${GOPATH}/src/github.com/GoogleContainerTools/kpt"
```

Download dependencies:

```sh
make tidy
```

Run; Porch is implemented as an extension k8s apiserver so to run it, we need:
* main apiserver
* etcd (to back the main apiserver)
* Porch (the extension apiserver)

But first we need to create docker network for all the containers to run on:

```sh
make network
```

```sh
# Start etcd
make start-etcd

# Start main apiserver
make start-kube-apiserver

# Start porch
make run-local

```

To teardown the Docker containers and network:

```sh
make stop
```

### Registering a Repository

Update the example configs of [git-repository.yaml](./config/samples/git-repository.yaml)
or [oci-repository.yaml](./config/samples/oci-repository.yaml)
with your Git repository or OCI repository respectively.

For Git:

* Create a Git repository for your blueprints.
* GitHub: Create a [Personal Access Token](https://github.com/settings/tokens) to use with Porch
* Create a secret with the token:
  ```sh
  kubectl create secret generic git-repository-auth \
    --namespace=default \
    --from-literal=username=<GitHub username> \
    --from-literal=token=<GitHub Personal Access Token>
  ```
* Update the [git-repository.yaml](./config/samples/git-repository.yaml) with your repository address
* Register the repository:
  ```sh
  KUBECONFIG=./hack/local/kubeconfig kubectl apply -f ./config/samples/git-repository.yaml
  ```

For OCI:

* Create an [Artifact Registry repository](https://console.cloud.google.com/artifacts)
* Update the [oci-repository.yaml](./config/samples/oci-repository.yaml) with your OCI repository address
* Make sure your application default credentials are up-to-date, i.e. by running:
  ```sh
  gcloud artifacts docker images list <your OCI repository address>
  ```
* Register the repository:
  ```sh
  KUBECONFIG=./hack/local/kubeconfig kubectl apply -f ./config/samples/oci-repository.yaml
  ```
 
List the package revisions:

```sh
export KUBECONFIG="$(pwd)/hack/local/kubeconfig"
kubectl get packagerevisions -oyaml
kubectl get packagerevisionresources -oyaml
```

Or create a pakcage revision:
```sh
kubectl apply -f ./config/samples/bucket-label.yaml
```

## Development

### Changing Types

If you change the API object type definitions in any of the
`api/porch/.../types.go`, update the generated code by running:

```sh
make generate
```

## Run in GKE Cluster

Prerequisite:
* Create GKE cluster
* Create appropriate KUBECONFIG.

### Build the Container Image

Build a Docker image using a script:

```sh
./hack/build-image.sh

# Supported flags
# --repository [REPO]      name of the Docker repository
# --project    [PROJECT]   GCP project (will translate to gcr.io/PROJECT)
# --tag [TAG]              image tag, i.e. 'latest'
# --push                   also push the image to the repository


# Example
./hack/build-image.sh --project=my-gcp-project --push
```

Or, build directly via docker:
**Note**: This must be done from the parent directory (kpt, not porch):

```sh
docker build -t TAG -f ./porch/hack/Dockerfile .
```

### Deploy into a Kubernetes Cluster

Edit `config/deploy/2-deployment.yaml`, updating the pod template's image
reference to match what you pushed and setting the `imagePullPolicy`
to something suitable.  Then call:

```sh
# Create CRDs
kubectl apply -f ./controllers/pkg/apis/porch/v1alpha1/
# Deploy Porch apiserver extension.
kubectl apply -f ./config/deploy/
```

When running you can:

```sh
# notice porch.kpt.dev/v1alpha1 in the result
kubectl api-resources

# List packagerevisions
kubectl get packagerevisions --namespace default
```

Follow the instructions above on how to register repositories and discover packges.

### Running Locally

Porch is an extension k8s apiserver. As such, it needs the main apiserver, which in turn needs `etcd`.

Start `etcd` and main apiserver:

```sh
make start-etcd
make start-kube-apiserver
```

Now, start the porch apiserver:

```sh
make run-local

# Call the server
KUBECONFIG=./hack/local/kubeconfig kubectl api-resources
# List package revisions
KUBECONFIG=./hack/local/kubeconfig kubectl get packagerevisions --namespace default
```
