---
title: "`push`"
linkTitle: "push"
type: docs
description: >
  Push resources to a package revision.
---

<!--mdtogo:Short
    Push resources to a package revision.
-->

`push` update the content of a package revision with
the provided resources.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg push PACKAGE_REV_NAME [DIR] [flags]
```

#### Args

```
PACKAGE_REV_NAME:
  The name of a an existing package revision in a repository.

DIR:
  A local directory with the new manifest. If not provided,
  the manifests will be read from stdin.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# update the package revision blueprint-f977350dff904fa677100b087a5bd989106d0456 with the resources
# in the ./package directory
$ kpt alpha rpkg push blueprint-f977350dff904fa677100b087a5bd989106d0456 ./package --namespace=default
```

<!--mdtogo-->