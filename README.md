# KPT Packaging Tool

Managed Kubernetes Configuration as *Data* using a GitOps workflow.

- Publish and Consume Configuration Packages -- using both public and private repos
- Merge upstream updates into local packages
- Works with both raw configuration packages (e.g. things that work with `kubectl apply -f`)
  and packages written in DSLs (e.g. Terraform)

## Community

**We'd love to hear from you!**

* [kpt-users mailing list](https://groups.google.com/forum/#!forum/kpt-users)

---------------------

[![Build Status](https://travis-ci.org/GoogleContainerTools/kpt.svg?branch=master)](https://travis-ci.org/GoogleContainerTools/kpt)
[![Code Coverage](https://codecov.io/gh/GoogleContainerTools/kpt/branch/master/graph/badge.svg)](https://codecov.io/gh/GoogleContainerTools/kpt)
[![Go Report Card](https://goreportcard.com/badge/GoogleContainerTools/kpt)](https://goreportcard.com/report/GoogleContainerTools/kpt)
[![LICENSE](https://img.shields.io/github/license/GoogleContainerTools/kpt.svg)](https://github.com/GoogleContainerTools/kpt/blob/master/LICENSE)
[![Releases](https://img.shields.io/github/release-pre/GoogleContainerTools/kpt.svg)](https://github.com/GoogleContainerTools/kpt/releases)

## Installation

Binaries:

- https://storage.cloud.google.com/kpt-dev/kpt.master_darwin_amd64
- https://storage.cloud.google.com/kpt-dev/kpt.master_linux_amd64
- https://storage.cloud.google.com/kpt-dev/kpt.master_windows_amd64

Containers:

- gcr.io/kpt-dev/kpt

From Source:

**Note: requires go 1.13 or higher**

```sh
# must be go 1.13 or higher
go version
```

```sh
GO111MODULE=on go get  github.com/GoogleContainerTools/kpt
```

```sh
# run the command
$(go env GOPATH)/bin/kpt help
```

## Documentation

All documentation is built directly into the command binary and can be accessed from the cli through
`kpt help`.

Built-in documentation has also been rendered as markdown files for friendly web viewing:
[docs](docs/README.md)

## Build from Source

```sh
# build the binary
make build

# generate code, perform linting, etc
make all
```

## Lead Developers

- Phillip Wittrock @pwittrock -- Kubernetes kubectl / sig-cli TL (Google)
