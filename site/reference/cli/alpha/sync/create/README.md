---
title: "`create`"
linkTitle: "create"
type: docs
description: >
  Create a new sync resource in the cluster.
---

<!--mdtogo:Short
    Create a new sync resource in the cluster.
-->

`create` adds a new RootSync resource to the cluster which references
the specified package. Config Sync then deploys the package into the cluster.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha sync create DEPLOYMENT_NAME [flags]
```

#### Args

```
DEPLOYMENT_NAME:
  The name of the sync resource created in the cluster.
```

#### Flags

```
--package
  Name of the package that should be deployed. It must exist in a
  deployment repo and be published.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# get a specific package in the default namespace
$ kpt alpha sync create my-app --package=deployment-8f9a0c7bf29eb2cbac9476319cd1ad2e897be4f9 --namespace=default
```

<!--mdtogo-->