---
title: "Container Runtime"
linkTitle: "Container Runtime"
weight: 1
type: docs
description: >
   Writing and running functions as containers
---

Functions may be written as container images. The container must run a program
which:

- Reads a ResourceList from STDIN
- Writes a ResourceList to STDOUT
- Exits non-0 on failure
- Writes error messages to users on STDERR

By default containers are run without network access, and without the ability
to write to volumes outside the container. All local environment variables
except the `TMPDIR` variable are loaded into the container by default. See the
[fn command reference] for more details.

While container functions may be written in any language so long as they
adhere to the [io specification] (read / write ResourceList), the
[kpt-functions-sdk] provides a solution for writing functions using typescript
and the [go libraries] provide utilities for writing them in golang.

## Imperative Run

Container functions may be run imperatively using the `--image` flag with
`kpt fn run`. This will create a container from the image, then write to its
STDIN a ResourceList created from the contents of the package directory, and
finally read from it's STDOUT a ResourceList used to write resource back to
the package directory.

```sh
kpt fn run DIR/ --image gcr.io/a/b:v1
```

## Declarative Run

Container functions may be run by declaring the function in a resource using
the `config.kubernetes.io/function` annotation.

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

To run the declared function use `kpt fn run DIR/` on the directory containing
the example.

## Next Steps

- Explore other ways to run functions from the [Functions Developer Guide].
- Find out how to structure a pipeline of functions from the
  [functions concepts] page.
- Consult the [fn command reference].

[io specification]: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md
[kpt-functions-sdk]: https://github.com/GoogleContainerTools/kpt-functions-sdk
[go libraries]: ../golang/
[Functions Developer Guide]: ../
[functions concepts]: ../../../../concepts/functions/
[fn command reference]: ../../../../reference/fn/
