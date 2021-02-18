---
title: "Packaging"
linkTitle: "Packaging"
weight: 3
type: docs
description: >
  Packaging goals and design decisions
---

## Packaging

The two primary sets of capabilities that are required to enable reuse are:

1. The ability to distribute/publish/share, compose, and update groups of
   configuration artifacts, commonly known as packages.
2. The ability to adapt them to your use cases, which we call customization.

In order to facilitate programmatic operations, kpt:

1. Relies upon git as the source of truth
2. Represents configuration as data, specifically represents Kubernetes object
   configuration as resources serialized in YAML or JSON format.

For compatibility with other arbitrary formats, kpt supports generating
resource configuration data from templates, configuration DSLs, and programs
using [source functions].

## Subpackages

A `subpackage` is a `kpt` package which is nested within the directory tree of
another `kpt` package.

Here is an example directory structure of a `kpt` package with subpackages

```sh
foo
├── Kptfile
├── bar # subpackage
│   ├── Kptfile
│   ├── baz # subpackage
│   │   ├── Kptfile
│   │   └── cm.yaml
│   └── deploy.yaml
└── service.yaml
```

#### Use cases

1. Package publishers need a way to pick and choose multiple component packages
   which work well together, create a single single `kpt` package using them to
   ship an out of the box application, maintain the package and abstract the
   details from package consumers. Alternatively, package consumers may [update]
   individual `subpackages` directly from the upstream sources.
2. Package publishers need a way to create parameter values (e.g. [setters]) to
   be consistent across multiple `subpackages` and make it easy for package
   consumers to provide them with a single command.
3. (Under development) Package consumers need a way to apply a set of
   `subpackages` in a single command to a live cluster while maintaining
   the ability to manage them (e.g. add/destroy) independently.

#### Principles

Here are the core principles around `subpackages` concept

1. Each kpt package is an independent building block and should contain resources
   (e.g. setter definitions) in its `Kptfile`.
2. Commands performed on a package are not performed on its subpackages unless
   `--recurse-subpackages(-R)` is provided. (only available with [cfg] commands currently
   and the default value of `-R` flag might vary for each command)

[source functions]: ../functions/#source-function
[update]: https://googlecontainertools.github.io/kpt/guides/consumer/update/
[setters]: https://googlecontainertools.github.io/kpt/guides/producer/setters/
[cfg]: https://googlecontainertools.github.io/kpt/reference/cfg/
