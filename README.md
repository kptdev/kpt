[![Build Status](https://travis-ci.org/GoogleContainerTools/kpt.svg?branch=master)](https://travis-ci.org/GoogleContainerTools/kpt)
[![Code Coverage](https://codecov.io/gh/GoogleContainerTools/kpt/branch/master/graph/badge.svg)](https://codecov.io/gh/GoogleContainerTools/kpt)
[![Go Report Card](https://goreportcard.com/badge/GoogleContainerTools/kpt)](https://goreportcard.com/report/GoogleContainerTools/kpt)
[![LICENSE](https://img.shields.io/github/license/GoogleContainerTools/kpt.svg)](https://github.com/GoogleContainerTools/kpt/blob/master/LICENSE)
[![Releases](https://img.shields.io/github/release-pre/GoogleContainerTools/kpt.svg)](https://github.com/GoogleContainerTools/kpt/releases)

# KPT Packaging Tool

Git based configuration package manager.

Publish, Consume and Update packages of raw Kubernetes Resource configuration.

## Installation

build using go:

    export GO111MODULE=on
    go install -v github.com/GoogleContainerTools/kpt
    go install -v sigs.k8s.io/kustomize/kustomize/v3

or download prebuilt binaries:

- [darwin](https://storage.cloud.google.com/kpt-dev/kpt.master_darwin_amd64)
- [linux](https://storage.cloud.google.com/kpt-dev/kpt.master_linux_amd64)
- [windows](https://storage.cloud.google.com/kpt-dev/kpt.master_windows_amd64)

## Quick Start: First 5 minutes

  Fetch a package from *any git repo* containing Kubernetes Resource configuration:

    $ kpt get https://github.com/kubernetes/examples/staging/cockroachdb my-cockroachdb

  Print the fetched package contents using kustomize:

    export KUSTOMIZE_ENABLE_ALPHA_COMMANDS=true # enable kustomize alpha commands
    kustomize config tree my-cockroachdb --name --image

  Output:

    my-cockroachdb
    ├── [cockroachdb-statefulset.yaml]  Service cockroachdb
    ├── [cockroachdb-statefulset.yaml]  StatefulSet cockroachdb
    │   ├── spec.replicas: 3
    │   └── spec.template.spec.containers
    │       └── 0
    │           ├── name: cockroachdb
    │           └── image: cockroachdb/cockroach:v1.1.0
    ├── [cockroachdb-statefulset.yaml]  PodDisruptionBudget cockroachdb-budget
    └── [cockroachdb-statefulset.yaml]  Service cockroachdb-public

  Apply the package to a cluster:

    kustomize apply my-cockroachdb/

## Whats next

See the full `kpt` [documentation](docs/README.md) or run `kpt help`

## FAQ

**Why Resource configuration rather than Templates or DSLs?**  Using Resource configuration
provides a number of desirable properties:

  1. it clearly **represents the intended state** of the infrastructure -- no for loops, http calls,
    etc to interpret

  2. it **works directly with Kubernetes project based tools** -- `kubectl`, `kustomize`, etc

  3. it enables **composition of a variety of tools written in different languages**
      * any modern language can manipulate yaml / json structures, no need to adopt `go`

  4. it **supports static analysis**
      * develop tools and processes to perform validation and linting

  5. it can be **modified programmatically**
      * develop CLIs and UIs for working with configuration rather than using `vim`

**Is there a container image that contains kpt?**

  Yes. [gcr.io/kpt-dev/kpt](Dockerfile) contains the `kpt` and `kustomize` binaries.

## Community

**We'd love to hear from you!**

* [kpt-users mailing list](https://groups.google.com/forum/#!forum/kpt-users)

---------------------
