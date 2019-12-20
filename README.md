[![Build Status](https://travis-ci.org/GoogleContainerTools/kpt.svg?branch=master)](https://travis-ci.org/GoogleContainerTools/kpt)
[![Code Coverage](https://codecov.io/gh/GoogleContainerTools/kpt/branch/master/graph/badge.svg)](https://codecov.io/gh/GoogleContainerTools/kpt)
[![Go Report Card](https://goreportcard.com/badge/GoogleContainerTools/kpt)](https://goreportcard.com/report/GoogleContainerTools/kpt)
[![LICENSE](https://img.shields.io/github/license/GoogleContainerTools/kpt.svg)](https://github.com/GoogleContainerTools/kpt/blob/master/LICENSE)
[![Releases](https://img.shields.io/github/release-pre/GoogleContainerTools/kpt.svg)](https://github.com/GoogleContainerTools/kpt/releases)

# KPT Packaging Tool

Git based configuration package manager.

Publish, Consume and Update packages of raw Kubernetes Resource configuration.

## Latest Binaries

Binaries:

- [https://storage.cloud.google.com/kpt-dev/kpt.master_darwin_amd64]
- [https://storage.cloud.google.com/kpt-dev/kpt.master_linux_amd64]
- [https://storage.cloud.google.com/kpt-dev/kpt.master_windows_amd64]

Containers:

- `gcr.io/kpt-dev/kpt`

## Quick Start

Instructions for the first 5 minutes...

    # download binaries or install using go (1.13 or later)
    GO111MODULE=on go get github.com/GoogleContainerTools/kpt
    GO111MODULE=on go get sigs.k8s.io/kustomize/kustomize/v3
    export KUSTOMIZE_ENABLE_ALPHA_COMMANDS=true # enable kustomize alpha commands

    # fetch a package from github
    kpt get https://github.com/kubernetes/examples/staging/cockroachdb my-cockroachdb

    # print the package contents
    kustomize config tree my-cockroachdb --name --image

    # apply the package to a cluster
    kubectl apply --recursive -f my-cockroachdb/

## Why Resource configuration for packages?

**Why Resource configuration rather than Templates or DSLs?**  Using Resource configuration
directly provides a number of desirable properties such as:

  - it clearly **represents the intended state** of the infrastructure -- no for loops, http calls,
    etc

  - it **works with Kubernetes project based tools**

  - it lends itself to the **development of new / custom tools**
    - new tools can be developed read and modify the package contents based on the Resource schema.
    - validation and linting tools (e.g. `kubeval`)
    - parsing and modifying via the cli (e.g. `kustomize config set`)
    - parsing and modifying declaratively through meta Resources
      (e.g. `kustomize`, `kustomize config run`)

  - tools may be written in **any language or framework**
    - tools just manipulate yaml / json directly, rather than manipulating Templates or DSLs
    - can use Kubernetes language libraries and openapi schema

## Whats next

See the full [documentation](docs/README.md)

Documentation also available in the `kpt` command by running `$ kpt help`

## Community

**We'd love to hear from you!**

* [kpt-users mailing list](https://groups.google.com/forum/#!forum/kpt-users)

---------------------



