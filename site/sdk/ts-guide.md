# Typescript SDK Developer Guide

This guide will walk you through developing a kpt function using the Typescript SDK.

## Setup

### System Requirements

Currently supported platforms: amd64 Linux/Mac

### Setting Up Your Local Environment

Currently supported platforms: amd64 Linux/Mac

- Install [kpt][download-kpt] and its dependencies
- Install [node][download-node]
  - The SDK requires `npm` version 6 or higher.
  - If installing node from binaries (i.e. without a package manager), follow
    these [installation instructions][install-node].

### Your Kubernetes Cluster

For the type generation functionality to work, you need a Kubernetes cluster
with the [CRD OpenAPI Publishing][crd-openapi] feature which is GA with
Kubernetes 1.16.

Alternatively, you can use an existing NPM package with pre-generated types
such as the `hello-world` package discussed in the [Quickstart] and skip to
[implementing the function](#implement-the-function).

### Working with CRDs

The SDK uses the k8s server to generate the Typescript classes. If your
function uses a Custom Resource Definition, make sure you apply it to the
cluster used for type generation:

```shell
$ kubectl apply -f /path/to/my/crd.yaml
```

## Create the NPM Package

To initialize a new NPM package, first create a package directory:

```shell
$ mkdir my-package
$ cd my-package
```

> **Note:** All subsequent commands are run from the `my-package/` directory.

Run the interactive initializer:

```shell
$ npm init kpt-functions
```

Follow the instructions and respond to all prompts.

This process will create the following:

1. `package.json`: The `kpt-functions` framework library is the only item
   declared in `dependencies`. Everything required to compile and test your
   config function is declared as `devDependencies`, including the
   `create-kpt-functions` CLI discussed later in the `README`.
1. `src/`: Directory containing the source files for all your functions, e.g.:
   - `my_func.ts`: Implement your function's interface here.
   - `my_func_test.ts`: Unit tests for your function.
   - `my_func_run.ts`: The entry point from which your function is run.
1. `src/gen/`: Contains Kubernetes core and CRD types generated from the
   OpenAPI spec published by the cluster you selected.
1. `build/`: Contains Dockerfile for each function, e.g.:
   - `my_func.Dockerfile`

Next, install all package dependencies:

```shell
$ npm install
```

In addition to installation, `install` compiles your function into the `dist/`
directory.

You can run your function directly:

```shell
$ node dist/my_func_run.js --help
```

Currently, it simply passes through the input configuration data. Let's remedy
this.

## Implement the Function

You can now start implementing the function using your favorite IDE, e.g.
[VSCode]:

```shell
$ code .
```

In `src/my_func.ts`, implement the `KptFunc` interface from the [TS SDK API].

Take a look at these [demo functions] to better understand how
to use the typescript library.

Once you've written some code, build the package with:

```shell
$ npm run build
```

Alternatively, run the following in a separate terminal. It will continuously
build your function as you make changes:

```shell
$ npm run watch
```

To run the tests, use:

```shell
$ npm test
```

## Debug and Test the Function

You may want to run a function developed with one of the config function SDKs
using the exec runtime in order to avoid the overhead associated with running
a container. To run your function in the exec runtime, you will first need to
package your function as an executable.

The below example shows how to run a typescript function using the kpt exec
runtime.

### Prerequisites

- Install the pkg CLI.

  ```shell
  $ npm install -g pkg
  ```

- Install your kpt-functions package module to create your function's
  distributable file.

  ```shell
  $ npm i
  ```

### Steps

1. Pass the path to the appropriate executable for your OS when running kpt
   using the exec runtime.

   ```shell
   $ kpt fn eval DIR --exec "node dist/my_func_run.js"
   ```

## Build and push container images

With your working function in-hand, it's time to package your function into an
executable container image.

To build the docker image:

```shell
$ npm run kpt:docker-build
```

You can now run the function container, e.g.:

```shell
$ kpt fn eval DIR --image gcr.io/kpt-functions-demo/my-func:dev
```

To push the image to your container registry of choice:

```shell
$ npm run kpt:docker-push
```

You'll need proper authentication/authorization to push to your registry.

`kpt:docker-push` pushes to the registry specified in the
`kpt.docker_repo_base` field in `package.json`. You can manually edit this
field at any time.

The default value for the container image tag is `dev`. This can be overridden
using`--tag` flag:

```shell
$ npm run kpt:docker-build -- --tag=latest
$ npm run kpt:docker-push -- --tag=latest
```

## Use the SDK CLI

The `create-kpt-functions` package (installed as `devDependencies`), provides
a CLI for managing the NPM package you created above. The CLI sub-commands can
be invoked via `npm run`. For example, to add a new function to the package:

```shell
$ npm run kpt:function-create
```

These sub-commands are available:

```
kpt:docker-create       Generate Dockerfiles for all functions. Overwrite
                        files if they exist.
kpt:docker-build        Build container images for all functions.
kpt:docker-push         Push container images to the registry for all
                        functions.
kpt:function-create     Generate stubs for a new function. Overwrites files
                        if they exist.
kpt:type-create         Generate classes for core and CRD types. Overwrite
                        files if they exist.
```

Flags are passed to the CLI after the `--` separator. For example, to pass a
tag when building a container image:

```shell
$ npm run kpt:docker-build -- --tag=latest
```

[download-kpt]: /book/01-getting-started/01-system-requirements
[download-node]: https://nodejs.org/en/download/
[install-node]: https://github.com/nodejs/help/wiki/Installation/
[install-node]: https://github.com/nodejs/help/wiki/Installation/
[install-docker]: https://docs.docker.com/engine/installation/
[crd-openapi]: https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.15.md#customresourcedefinition-openapi-publishing
[quickstart]: ../quickstart/
[vscode]: https://code.visualstudio.com/
[ts sdk api]: https://googlecontainertools.github.io/kpt-functions-sdk/api/
[demo functions]: https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src/
