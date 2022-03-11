# Package Orchestration Server

Package Orchestration Server (a.k.a. Porch) is a k8s extension apiserver
which manages lifecycle of KRM configuration packages.

The two common ways to run Porch is:

* [On GKE](./docs/running-on-gke.md)
* [Locally](./docs/running-locally.md), intended primarily for [development](./docs/developing.md)

# Prerequisites

To use Porch, you will need:

* `kubectl`
* `gcloud` (if running on GKE)


# Using Porch

Make sure that your `kubectl` context is set up for `kubectl` to interact with the
correct Kubernetes instance (see instructions in [running-on-gke](./docs/running-on-gke.md)
or [running-locally](./docs/running-locally.md).

To check whether `kubectl` is configured with your Porch cluster (or local instance), run:

```sh
kubectl api-resources | grep porch
```

You should see the following four resourceds listed:

```
repositories                  config.porch.kpt.dev/v1alpha1          true         Repository
packagerevisionresources      porch.kpt.dev/v1alpha1                 true         PackageRevisionResources
packagerevisions              porch.kpt.dev/v1alpha1                 true         PackageRevision
functions                     porch.kpt.dev/v1alpha1                 true         Function
```

## Porch Resources

Porch manages four k8s resources:

1. `repositories`: Repository (Git or OCI) can be registered with Porch to support discovery
   or management of KRM configuration packages in those repositories, or discovery of KRM
   functions in those repositories.
2. `packagerevisions`: PackageRevision represents a revision of a KRM configuration package
   managed by Porch in one of the registered repositories (Git or OCI). The `PackageRevision`
   resource presents a _metadata_ view of the KRM configuration package.
3. `packagerevisionresources`: PackageRevisionResources represents an alterative _view_ of the
   same KRM configuration package - the actual configuration content, or KRM resources contained
   in the package.
4. `functions`: Function resource represents a KRM function discovered in a repository
   registered with Porch. When repository is registered with Porch, registration metadata
   can indicate that the OCI repository contains functions. Functions are only supported
   with OCI repositories.

Note that `PackageRevision` and `PackageRevisionResources` present different _views_ of the same
underlying KRM configuration package. `PackageRevision` represents the package metadata,
`PackageRevisionResources` represents the package content. The matching resources share the same
`name` (as well as API group and version: `porch.kpt.dev/v1alpha1`) and differ in kind
(`PackageRevision` and `PackageRevisionResources` respectively).

## Registering a Repository

To register a package repository with Porch, you can start with the example configs:
* [git-repository.yaml](./config/samples/git-repository.yaml), or
* [oci-repository.yaml](./config/samples/oci-repository.yaml)

These examples show how to create the Repository resource for your Git repository or
OCI repository respectively.

### Registering Git Repository

* Create a Git repository for your blueprints.
* GitHub: Create a [Personal Access Token](https://github.com/settings/tokens) to use with Porch
* Create a Kubernetes `Secret` resource to store repository authentication information.
  Example (**please make sure to update the `namespace` and `name` to match your environment):
  ```sh
  kubectl create secret generic git-repository-auth \
    --namespace=default \
    --type=kubernetes.io/basic-auth \
    --from-literal=username=<GitHub username> \
    --from-literal=password=<GitHub Personal Access Token>
  ```
* Update the [git-repository.yaml](./config/samples/git-repository.yaml) with your repository address,
  for example: `https://githug.com/my-organization/my-repository.git`
* Register the repository:
  ```sh
  kubectl apply -f ./config/samples/git-repository.yaml
  ```

Alternatively, you can use `kpt alpha repo register` feature:

```sh
# Register a repository using kpt
kpt alpha repo register https://github.com/platkrm/demo-blueprints.git --namespace default
```

### Registering an OCI Repository:

* Create an [Artifact Registry repository](https://console.cloud.google.com/artifacts)
* Update the [oci-repository.yaml](./config/samples/oci-repository.yaml) with your OCI repository address
* (If running Porch locally) Make sure your application default credentials are up-to-date,
  i.e. by running:
  ```sh
  gcloud artifacts docker images list <your OCI repository address>
  ```
* Register the repository with Porch
  ```sh
  kubectl apply -f ./config/samples/oci-repository.yaml
  ```

When running on GKE, Porch will authenticate with the OCI repository using the GCP service account
`iam.gke.io/gcp-service-account=porch-server@$(GCP_PROJECT_ID).iam.gserviceaccount.com`. Please make
sure that the service account has appropriate level of access to your OCI repository.

## Package Discovery

Once you have one or more repositories registered, you can list the package revisions:

```sh
export KUBECONFIG="${PWD}/hack/local/kubeconfig"

# List all package revisions
kubectl get packagerevisions

# Get metadata for a specific package
kubectl get packagerevisions --namespace default demo-blueprints:basens:v1 -oyaml

# or, using kpt
kpt alpha rpkg get --namespace default demo-blueprints:basens:v1 -oyaml

# Get resources
kubectl get packagerevisionresources --namespace default demo-blueprints:basens:v1 -oyaml

# Or, get the package resources as ResourceList using kpt:
kpt alpha rpkg pull --namespace default demo-blueprints:basens:v1

```

## Note on Resource Names

The names of `PackageRevision` and `PackageRevisionResources` resources consist of three
colon-separated parts:

`<repository name>:<package name>:<package version>`

## Package Authoring

### Initialize a New Package

To create a new package revision, start with [new-package.yaml](./config/samples/new-package.yaml).
Update the package name to match your registered repository.
Porch server creates and initializes the package and saves it (as a draft) into the registered repository.

```sh
# Initialize a new package; make sure to update the file first!
kubectl apply -f ./config/samples/new-package.yaml

# Inspect the package resources
kubectl get packagerevisionsresources --namespace default <resource name> -oyaml
```

Or, using `kpt alpha` features:

```sh
# Initialize a new package
kpt alpha rpkg init \
  --namespace default \
  --description "New Package Description" \
  --keywords="example,package" \
  --site="https://kpt.dev/blueprint" \
  blueprints:new-package:v0

# Inspect the package resources
kpt alpha rpkg pull --namespace default blueprints:new-package:v0

```

### Clone a Package

To create a new package revision by cloning another (upstream) package, you can start by
updating the [bucket-label.yaml](./config/samples/bucket-label.yaml) sample.
Make sure to update the resource name. The `bucket-label.yaml` example demonstrates
creation of package by cloning an upstream package and then evaluating a function
on the package resources.

```sh
# Create a package revision by cloning an upstream package
kubectl apply -f ./config/samples/bucket-label.yaml
```

Or, using `kpt alpha` feature (note that `kpt alpha rpkg clone` only clones the package;
it doesn't support subsequently evaluating a kpt function at this time:

```sh
kpt alpha rpkg clone \
  --strategy=resource-merge \
  --namespace=default \
  test-blueprints:basens:v1 \
  blueprints:cloned-package:v0
```

### Update Package Resources

To update package resources, you can get the package's `PackageRevisionResources`, and
update the resource with Porch. This will update the package's resources.

```sh
# Get the package resources
kubectl get packagerevisionresources -oyaml --namespace default \
  blueprints:cloned-package:v0 > resource.yaml

# Edit the resources using your favorite editor

# Update the resources
kubectl replace packagerevisionresources -oyaml --namespace default \
  blueprints:cloned-package:v0 -f resource.yaml
```

Or, using `kpt alpha`:
```sh
# Get the package resources and save them in a directory
kpt alpha rpkg pull --namespace default blueprints:cloned-package:v0 ./package

# Edit using your favorite editor

# Update the package resources:
kpt alpha rpkg push --namespace default blueprints:cloned-package:v0 ./package
```

### Unregister a Repository

To unregister a repository, delete the `Repository` resource:

Using `kubectl`:

```sh
kubectl delete -f ./config/samples/git-repository.yaml
```

Alternatively, you can use `kpt alpha repo unregister` feature:

```sh
# Register a repository using kpt
kpt alpha repo unregister demo-blueprints --namespace default
```
