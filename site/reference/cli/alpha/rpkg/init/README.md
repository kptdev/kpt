---
title: "`init`"
linkTitle: "init"
type: docs
description: >
  Initializes a new package in a repository.
---

<!--mdtogo:Short
    Initializes a new package in a repository.
-->

`init` initializes a new package in a repository. The inital package revision
will be empty.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg init PACKAGE_NAME [flags]
```

#### Args

```
PACKAGE_NAME:
  The name of the new package.
```

#### Flags

```
--repository
  Repository in which the new package will be created.

--revision
  Revision of the new package. The default value if v1.

--description
  short description of the package

--keywords
  list of keywords for the package

--site
  link to page with information about the package
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# create a new package named foo in the repository blueprint.
$ kpt alpha rpkg init foo --namespace=default --repository=blueprint
```

<!--mdtogo-->