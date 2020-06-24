---
title: "TypeScript Function SDK"
linkTitle: "TypeScript Function SDK"
weight: 5
type: docs
description: >
   Writing functions in TypeScript.
---

We provide an opinionated Typescript SDK for implementing config functions. This provides various
advantages:

- **General-purpose language:** Domain-Specific Languages begin their life with a reasonable
  feature set, but often grow over time. They bloat in order to accommodate the tremendous variety
  of customer use cases. Rather than follow this same course, config functions employ a true,
  general-purpose programming language that provides:
  - Proper abstractions and language features
  - A extensive ecosystem of tooling (e.g. IDE support)
  - A comprehensive catalog of well-supported libraries
  - Robust community support and detailed documentation
- **Type-safety:** Kubernetes configuration is typed, and its schema is defined using the OpenAPI spec.
  Typescript has a sophisticated type system that accommodates the complexity of Kubernetes resources.
  The SDK enables generating Typescript classes for core and CRD types, providing safe and easy
  interaction with Kubernetes objects.
- **Batteries-included:** The SDK provides a simple, powerful API for querying and manipulating configuration
  files. It provides the scaffolding required to develop, build, test, and publish functions,
  allowing you to focus on implementing your business-logic.

## Developer Quickstart

This quickstart will get you started developing a config function with the TypeScript SDK,
using an existing Hello World package.

### Prerequisites

#### System Requirements

Currently supported platforms: amd64 Linux/Mac

#### Setting Up Your Local Environment

- Install [node][download-node]
- Install [docker][install-docker]
- Install [kpt][download-kpt] and add it to \$PATH

### Hello World Package

1. Get the `hello-world` package:

   ```sh
   git clone --depth 1 https://github.com/GoogleContainerTools/kpt-functions-sdk.git
   ```

   All subsequent commands are run from the `hello-world` directory:

   ```sh
   cd kpt-functions-sdk/ts/hello-world
   ```

1. Install all dependencies:

   ```sh
   npm install
   ```

1. Run the following in a separate terminal to continuously build your function as you make changes:

   ```sh
   npm run watch
   ```

1. Run the [`label_namespace`][label-namespace] function:

   ```sh
   export CONFIGS=../../example-configs

   kpt fn source $CONFIGS |
   node dist/label_namespace_run.js -d label_name=color -d label_value=orange |
   kpt fn sink $CONFIGS
   ```

   As the name suggests, this function added the given label to all `Namespace` objects
   in the `example-configs` directory:

   ```sh
   git diff $CONFIGS
   ```

1. Try modifying the function in `src/label_namespace.ts` to perform other operations
   on `example-configs`, then repeat step 4. You can also explore other functions such as [suggest-psp] or [validate-rolebinding].

   The function should implement the `KptFunc` interface [documented here][api-kptfunc].

## Next Steps

- Take a look at [these example functions][demo-funcs] to better understand how to use the typescript SDK.
- Read the complete [Typescript Developer Guide].
- Learn how to [run functions].

[download-node]: https://nodejs.org/en/download/
[install-docker]: https://docs.docker.com/v17.09/engine/installation
[download-kpt]: ../../../../installation/
[demo-funcs]: https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src
[api-kptfunc]: https://googlecontainertools.github.io/kpt-functions-sdk/api/interfaces/_types_.kptfunc.html
[Typescript Developer Guide]: develop/
[run functions]: ../../../consumer/function/
[label-namespace]: https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src/label_namespace.ts
[suggest-psp]: https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src/suggest_psp.ts
[validate-rolebinding]: https://github.com/GoogleContainerTools/kpt-functions-sdk/tree/master/ts/demo-functions/src/validate_rolebinding.ts
