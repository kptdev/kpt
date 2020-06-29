---
title: "Exec Runtime"
linkTitle: "Exec Runtime"
weight: 2
type: docs
description: >
   Writing and running functions as executables
---

Functions may alternatively be run as executables outside of containers.  Exec
functions read input and write output the same as container functions, but are
run outside of a container.

Running functions as executables can be useful for function development, or for
running trusted executables.

Exec functions are disabled by default, and must be enabled with `--enable-exec`.

{{% pageinfo color="info" %}}
Exec functions may be converted to container functions by building the executable
into a container and invoking it as the container `CMD`.
{{% /pageinfo %}}

## Imperative Run

Exec functions may be run imperatively using the `--exec-path` flag with `kpt fn run`.

```sh
kpt fn run DIR/ --enable-exec --exec-path /path/to/executable
```

This is similar to building `/path/to/executable` into the container image
`gcr.io/project/image:tag` and running -- except that the executable has access
to the local machine.

```sh
kpt fn run DIR/ --image gcr.io/project/image:tag
```

Just like container functions, exec functions accept input as arguments after `--`

```sh
kpt fn run DIR/ --enable-exec --exec-path /path/to/executable -- foo=bar
```

## Declarative Run

Exec functions may also be run by declaring the function in a resource using the
`config.kubernetes.io/function` annotation.

To run the declared function use `kpt fn run DIR/ --enable-exec` on the directory containing
the example.

```yaml
apiVersion: example.com/v1beta1
kind: ExampleKind
metadata:
  name: function-input
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: /path/to/executable
```

Note: if the `--enable-exec` flag is not provided, `kpt fn run DIR/` will ignore the exec
function and exit 0.

## Typescript Function Example

You may want to run a function developed with one of the config function SDKs using the exec
runtime in order to avoid the overhead associated with running a container. To run your function
in the exec runtime, you will first need to package your function as an executable.

We walk through an example of running a typescript function using the kpt exec runtime.

### Prerequisites

* Install the pkg CLI.

    ```sh
    npm install -g pkg
    ```

* Install your kpt-functions package module to create your function's distributable file.

    ```sh
    npm i
    ```

### Steps

1. Use the pkg CLI to create an executable from your function's distributable file. For a my_func
   function built using the typescript SDK, this is `dist/my_func_run.js`.

    ```sh
    npx pkg dist/my_func_run.js
    ```

2. Pass the path to the appropriate executable for your OS when running kpt using the exec runtime.

    ```sh
    kpt fn run DIR/ --enable-exec --exec-path /path/to/my_func_run-macos -- foo=bar baz=qux
    ```

## Next Steps

* Find out how to structure a pipeline of functions from the [functions concepts] page.
* Consult the [fn command reference].

[functions concepts]: ../../../../concepts/functions/
[fn command reference]: ../../../../reference/fn/