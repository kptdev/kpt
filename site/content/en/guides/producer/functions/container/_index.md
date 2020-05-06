---
title: "Container Runtime"
linkTitle: "Container Runtime"
weight: 1
type: docs
description: >
   Writing and running functions as containers
---

Functions may be written as container images.  The container must run a program which:

- Reads a ResourceList from STDIN
- Writes a ResourceList to STDOUT
- Exits non-0 on failure
- Writes error messages to users on STDERR

By default containers are run without network access, and without the ability to write to volumes
outside the container.

While container functions may be written in any language so long as they adhere to the io
specification (read / write ResourceList), the [kpt-functions-sdk] provides a solution for
writing functions using typescript.

## Imperative Run

Container functions may be run imperatively using the `--image` flag with `kpt fn run`.  This
will create a container from the image, then write to its STDIN a ResourceList and read from
it's STDOUT a ResourceList.

## Declarative Run

Container functions may be run by declaring the function in a resource using the
`config.kubernetes.io/function` annotation.

```yaml
apiVersion: example.com/v1beta1
kind: ExampleKind
metadata:
  name: function-input
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gcr.io/a/b:v1
```

To run the declared function use `kpt fn run DIR/` on the directory containing the example.

## Network

By default, container functions cannot access network or volumes.  Functions may enable network
access using the `--network` flag, and specifying that a network is required in the functionConfig.

```yaml
apiVersion: example.com/v1beta1
kind: ExampleKind
metadata:
  name: function-input
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gcr.io/a/b:v1
        network:
          required: true
```

```shell script
kpt fn run DIR/ --network
```

[kpt-functions-sdk]: https://github.com/GoogleContainerTools/kpt-functions-sdk