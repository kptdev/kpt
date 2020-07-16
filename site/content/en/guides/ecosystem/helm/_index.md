---
title: "Helm"
linkTitle: "Helm"
weight: 2
type: docs
description: >
    Generate kpt packages from Helm charts
---

*Any solution which emits configuration can also generate kpt packages
(because they are just configuration).*

Helm charts may be used to generate kpt packages which can then be further
customized directly.

## Steps

1. [Fetch a Helm chart](#fetch-a-helm-chart)
2. [Expand the Helm chart](#expand-the-helm-chart)
3. [Publish the kpt package](#publish-the-kpt-package)

## Fetch a Helm chart

##### Command

```sh
helm fetch stable/mysql
```

Pull down a helm chart for expansion.  This may optionally be checked into
git so it can be expanded again in the future.

## Expand the Helm chart

##### Command

```sh
helm template mysql-1.3.1.tgz --output-dir .
```

Expand the Helm chart into resource configuration.  Template values may be
provided on the commandline or through a `value.yaml`

##### Output

```sh
wrote ./mysql/templates/secrets.yaml
wrote ./mysql/templates/tests/test-configmap.yaml
wrote ./mysql/templates/pvc.yaml
wrote ./mysql/templates/svc.yaml
wrote ./mysql/templates/tests/test.yaml
wrote ./mysql/templates/deployment.yaml
```

##### Command

```sh
tree mysql/
```

##### Output

```
mysql
└── templates
├── deployment.yaml
├── pvc.yaml
├── secrets.yaml
├── svc.yaml
└── tests
├── test-configmap.yaml
└── test.yaml
```

## Publish the kpt package

The expanded chart will function as a kpt package once checked into a git
repository.  It may optionally be tagged with a package version.

```sh
git add .
git commit -m “add mysql package”
git tag package-examples/mysql/mysql/templates/v0.1.0
git push package-examples/mysql/mysql/templates/v0.1.0
```

Once stored in git, kpt can be used to fetch the package and customize it directly.

```sh
export REPO=https://github.com/GoogleContainerTools/kpt.git
kpt pkg get $REPO/package-examples/mysql/mysql/templates@v0.16.0 mysql/
```

The package local package can be modified after it is fetched, and pull in
upstream changes when the upstream package is regenerated from the chart
or otherwise modified.
