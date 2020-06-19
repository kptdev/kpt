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

Let’s look at the example of imperatively running a function to set a label
value.  The ([label-namespace]) image contains a program which adds a label to all Namespace resources
provided to it.

```sh
kpt fn run --image gcr.io/kpt-functions/label-namespace . -- label_name=color label_value=orange
```

Kpt read the resources from the package at “.”, provided them as input to
a container running `gcr.io/kpt-functions/label-namespace`, and wrote the
resources back to the package.

Functions can address many types of workflows, including:

- Generating resources from some inputs (like client-side CRDs)
- Applying cross-cutting transformations (e.g., set a field on all resources that
  look like this)
- Validating resources conform to best practices defined by the organization
  (e.g., must specify tag as part of the image)
- Sending resources to a destination (e.g., saving them locally or deploying them to a cluster)

## Next Steps

- See more examples of functions in the [functions catalog].
- Get a quickstart on writing functions from the [function producer docs].
- Find out how to structure a pipeline of functions from the [functions concepts] page.

[label-namespace]: https://github.com/GoogleContainerTools/kpt-functions-sdk/blob/master/ts/hello-world/src/label_namespace.ts
[functions catalog]: catalog/
[function producer docs]: ../../producer/functions/
[functions concepts]: ../../../concepts/functions/
