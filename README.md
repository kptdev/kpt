# KPT Packaging Tool

Managed Kubernetes Configuration as *Data* using a GitOps workflow.

- Publish and Consume Configuration Packages -- using both public and private repos
- Customize local package copies
- Merge upstream updates into local packages
- Works with both raw configuration packages (e.g. kubectl apply-able packages) and packages
  generated from DSLs (e.g. Helm charts)

## Community

**Important:** KPT has not publicly launched, and is confidential to a group of whitelisted EAP
users. Group membership at this time is invite only.

<!-- TODO: add a kubernetes slack channel after we launch publicly -- could just be sig-cli -->

**We'd love to hear from you!**

* [kpt-users mailing list](https://groups.google.com/forum/#!forum/kpt-users)

---------------------

[![Build Status](https://travis-ci.org/GoogleContainerTools/kpt.svg?branch=master)](https://travis-ci.org/GoogleContainerTools/kpt)
[![Code Coverage](https://codecov.io/gh/GoogleContainerTools/kpt/branch/master/graph/badge.svg)](https://codecov.io/gh/GoogleContainerTools/kpt)
[![Go Report Card](https://goreportcard.com/badge/GoogleContainerTools/kpt)](https://goreportcard.com/report/GoogleContainerTools/kpt)
[![LICENSE](https://img.shields.io/github/license/GoogleContainerTools/kpt.svg)](https://github.com/GoogleContainerTools/kpt/blob/master/LICENSE)
[![Releases](https://img.shields.io/github/release-pre/GoogleContainerTools/kpt.svg)](https://github.com/GoogleContainerTools/kpt/releases)

## Installation

<!-- TODO: change this to `go get kpt.dev@0.1.0` when the domain is setup and the repo is public-->

```sh
git clone git@github.com:GoogleContainerTools/kpt
cd kpt/cmd
go build ./
```

## Documentation

All documentation is built directly into the command binary and can be accessed from the cli through
`kpt help COMMAND` or `kpt tutorials TUTORIAL_NAME`.

Built-in documentation has also been rendered as markdown files for friendly web viewing:

- Tutorials: [tutorials](docs/tutorials.md)
- Command Reference: [docs](docs/kpt.md)
- Package Duck-Typed Command Reference: [docs](docs/kpt_duck-typed.md)
- API Reference: [docs](docs/apis.md)

## Lead Developers

- Phillip Wittrock @pwittrock -- Kubernetes kubectl / sig-cli TL (Google)
- Sunil Arora @droot -- Kubernetes kubebuilder / controller-runtime TL (Google)
- Jeff Regan @monopole -- Kubernetes kustomize TL (Google)
