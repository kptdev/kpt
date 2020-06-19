---
title: "TypeScript Developer Guide"
linkTitle: "TypeScript Developer Guide"
weight: 6
type: docs
description: >
   Developing functions in TypeScript.
---

This guide will walk you through developing a KPT function using the Typescript SDK.

## Setup

### System Requirements

Currently supported platforms: amd64 Linux/Mac

### Setting Up Your Local Environment

- Install [node][download-node]
  - The SDK requires `npm` version 6 or higher.
  - If installing node from binaries (i.e. without a package manager), follow these
    [installation instructions][install-node].
- Install [docker][install-docker]

### Your Kubernetes Cluster

For the type generation functionality to work, you need a Kubernetes cluster with the [CRD OpenAPI Publishing][beta-feature] feature which is beta with Kubernetes 1.15.

Alternatively, you can use an existing NPM package with pre-generated types such as the `hello-world`
package discussed in the [Quickstart](../#developer-quickstart) and skip to [implementing the function](#implement-the-function).

#### Using a `kind` Cluster

The easiest way to get developing is to use `kind` to bring up a cluster running in a local container.

1. Download the [kind binary][download-kind] version 0.5.1 or higher
1. Use this config file:

   ```sh
   cat > kind.yaml <<EOF
   kind: Cluster
   apiVersion: kind.sigs.k8s.io/v1alpha3
   kubeadmConfigPatches:
   - |
     apiVersion: kubeadm.k8s.io/v1beta2
     kind: ClusterConfiguration
     metadata:
       name: config
     apiServer:
       extraArgs:
         "feature-gates": "CustomResourcePublishOpenAPI=true"
   nodes:
   - role: control-plane
   EOF
   ```

   Note the use of the beta feature.

1. Create the cluster:

   ```sh
   kind create cluster --name=kpt-functions --config=kind.yaml --image=kindest/node:v1.15.7
   ```

#### Using a GKE cluster

You can also use a deployed cluster in GKE. The beta k8s feature is avilable only when using GKE's
`--enable-kubernetes-alpha` flag, as seen here:

```sh
gcloud container clusters create $USER-1-15 --cluster-version=latest --region=us-central1-a --project <PROJECT>
gcloud container clusters get-credentials $USER-1-15 --zone us-central1-a --project <PROJECT>
```

### Working with CRDs

The SDK uses the k8s server to generate the typescript classes.
If your function uses a Custom Resource Definition, make sure you apply it to the cluster used
for type generation:

```sh
kubectl apply -f /path/to/my/crd.yaml
```

## Create the NPM package

To initialize a new NPM package, first create a package directory:

```sh
mkdir my-package
cd my-package
```

> **Note:** All subsequent commands are run from the `my-package/` directory.

Run the interactive initializer:

```sh
npm init kpt-functions
```

Follow the instructions and respond to all prompts.

This process will create the following:

1. `package.json`: The `kpt-functions` framework library is the only item declared in `dependencies`.
   Everything required to compile and test your KPT function is declared as `devDependencies`,
   including the `create-kpt-functions` CLI discussed later in the `README`.
1. `src/`: Directory containing the source files for all your functions, e.g.:

   - `my_func.ts`: Implement your function's interface here.
   - `my_func_test.ts`: Unit tests for your function.
   - `my_func_run.ts`: The entry point from which your function is run.

1. `src/gen/`: Contains Kubernetes core and CRD types generated from the OpenAPI spec published by the cluster you selected.
1. `build/`: Contains Dockerfile for each function, e.g.:
   - `my_func.Dockerfile`

Next, install all package dependencies:

```sh
npm install
```

In addition to installation, `install` compiles your function into the `dist/` directory.

You can run your function directly:

```sh
node dist/my_func_run.js --help
```

Currently, it simply passes through the input configuration data. Let's remedy this.

## Implement the function

You can now start implementing the function using your favorite IDE, e.g. [VSCode][vscode]:

```sh
code .
```

In `src/my_func.ts`, implement the `KptFunc` interface [documented here][api-kptfunc].

Take a look at [these example functions][demo-funcs] to better understand how to use
the typescript library. These functions are available as container images documented in the [catalog][catalog].

Once you've written some code, build the package with:

```sh
npm run build
```

Alternatively, run the following in a separate terminal. It will continuously build your function
as you make changes:

```sh
npm run watch
```

To run the tests, use:

```sh
npm test
```

## Build and push container images

With your working function in-hand, it's time to package your function into an executable container
image.

To build the docker image:

```sh
npm run kpt:docker-build
```

You can now run the function container, e.g.:

```sh
docker run gcr.io/kpt-functions-demo/my-func:dev --help
```

To push the image to your container registry of choice:

```sh
npm run kpt:docker-push
```

You'll need proper authentication/authorization to push to your registry.

`kpt:docker-push` pushes to the registry specified in the `kpt.docker_repo_base` field in `package.json`.
You can manually edit this field at any time.

The default value for the container image tag is `dev`. This can be overridden using`--tag` flag:

```sh
npm run kpt:docker-build -- --tag=latest
npm run kpt:docker-push -- --tag=latest
```

## Use the SDK CLI

The `create-kpt-functions` package (installed as `devDependencies`), provides a CLI for managing
the NPM package you created above. The CLI sub-commands can be invoked via `npm run`. For example,
to add a new function to the package:

```console
npm run kpt:function-create
```

These sub-commands are available:

```console
kpt:docker-create       Generate Dockerfiles for all functions. Overwrite
                        files if they exist.
kpt:docker-build        Build container images for all functions.
kpt:docker-push         Push container images to the registry for all functions.
kpt:function-create     Generate stubs for a new function. Overwrites files
                        if they exist.
kpt:type-create         Generate classes for core and CRD types. Overwrite
                        files if they exist.
```

Flags are passed to the CLI after the `--` separator. For example, to pass a tag
when building a container image:

```console
npm run kpt:docker-build -- --tag=latest
```

## Next Steps

- Learn how to [run functions].
- Find out how to structure a pipeline of functions from the [functions concepts] page.
- Take a look at [these example functions][demo-funcs] to better understand how to use the typescript SDK.

[spec]: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md
[kustomize-run]: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-impl.md
[demo-funcs]: https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src
[label-namespace]: https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src/label_namespace.ts
[catalog]: ../../../../consumer/function/catalog
[api-kptfunc]: https://googlecontainertools.github.io/kpt-functions-sdk/api/interfaces/_types_.kptfunc.html
[vscode]: https://code.visualstudio.com/
[npm-packages]: https://github.com/GoogleContainerTools/kpt-functions-sdk/packages
[download-node]: https://nodejs.org/en/download/
[download-kind]: https://github.com/kubernetes-sigs/kind
[install-node]: https://github.com/nodejs/help/wiki/Installation
[install-docker]: https://docs.docker.com/v17.09/engine/installation
[beta-feature]: https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.15.md#customresourcedefinition-openapi-publishing
[run functions]: ../../../../consumer/function/
[functions concepts]: ../../../../../concepts/functions/
