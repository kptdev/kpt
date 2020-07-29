---
title: "Running functions"
linkTitle: "Running Functions"
weight: 7
type: docs
description: >
    Modify or validate the contents of a package by calling a function.
---

## Functions User Guide

When an operation requires more than just the schema, and data is necessary,
the dynamic logic can be built into a separate tool.
Functions bundle dynamic logic in container images and apply that logic to the
contents of a package -- modifying and validating package contents.

{{% pageinfo color="primary" %}}
Functions provide a common interface for writing programs to read and write
resources as data. This enables greater reuse and composition than when
configuration is itself represented as code. Functions may be written in any
language, or simply wrap other existing programs.
{{% /pageinfo %}}

{{< svg src="images/fn" >}}

Functions can address many types of workflows, including:

- Generating resources from some inputs (like client-side CRDs)
- Applying cross-cutting transformations (e.g., set a field on all resources
  that look like this)
- Validating resources conform to best practices defined by the organization
  (e.g., must specify tag as part of the image)
- Sending resources to a destination (e.g., saving them locally or deploying
  them to a cluster)

## Calling Functions Imperatively

Let’s look at the example of imperatively running a function to set a label
value.  The ([label-namespace]) image contains a program which adds a label to
all Namespace resources provided to it.

```sh
kpt fn run --image gcr.io/kpt-functions/label-namespace . -- label_name=color label_value=orange
```

Kpt read the configs from the package at “.” to generate input resources.
Behind the scenes, it also parsed the arguments and provided them through a
functionConfig field along with the input. It passed this information to a
container running `gcr.io/kpt-functions/label-namespace`, and wrote the
resources back to the package.

## Calling Functions Declaratively

The most common way of invoking config functions in production will be the
[declarative method].

Let's run the same [label-namespace] function declaratively, which means we
make a reusable function configuration resource which contains all information
necessary to run the function, from container image to argument values. Once we
create file with this information we can check it into [VCS] and run the
function in a repeatable fashion, making it incredibly powerful for production
use.

First create a function configuration file in the directory you want to apply
the function e.g. `label-ns-fc.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gcr.io/kpt-functions/label-namespace
data:
  "label_name": "color"
  "label_value": "orange"
```

This file contains a `config.kubernetes.io/function` annotation specifying the
docker image to use for the config as well as a data field containing a
key-value map of the "label_name" and "label_value" arguments specified
earlier. Using a map also makes it easier to pass more complex arguments values
like a list of strings.

Next, run the function.

```sh
kpt fn run .
```

Kpt used configs from the package at “.” to generate input resources. It
recognized that the `config.kubernetes.io/function` annotation denoted
`label-ns-fc.yaml` as a function config file and ran a container using the
`gcr.io/kpt-functions/label-namespace` image, passing in the function config
resource as well as all other input resources.

## Next Steps

- See more examples of functions in the [functions catalog].
- Get a quickstart on writing functions from the [function producer docs].
- Find out how to structure a pipeline of functions from the
  [functions concepts] page.
- Learn more ways of using the `kpt fn` command from the [reference] doc.

[label-namespace]: https://github.com/GoogleContainerTools/kpt-functions-sdk/blob/master/ts/hello-world/src/label_namespace.ts
[functions catalog]: catalog/
[function producer docs]: ../../producer/functions/
[functions concepts]: ../../../concepts/functions/
[declarative method]: ../../../reference/fn/run/#declaratively-run-one-or-more-functions
[reference]: ../../../reference/fn/run/
[VCS]: https://en.wikipedia.org/wiki/Version_control
