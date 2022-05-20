---
title: "`copy`"
linkTitle: "copy"
type: docs
description: >
  Create a new package revision from an existing one.
---

<!--mdtogo:Short
    Create a new package revision from an existing one.
-->

`copy` creates a new package revision from an existing one. The new
revision will be identical to the existing one but with a different
revision.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg copy SOURCE_PACKAGE_REV_NAME [flags]
```

#### Args

```
SOURCE_PACKAGE_REV_NAME:
  The name of the package revision that will be used as the source
  for creating a new package revision.
```

#### Flags

```
--revision
  Revision for the new package. If this is not specified, the default
  revision will be `latest + 1`. The default can only be used if the
  latest package revision is of the format `^v[0-9]+$`.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# create a new package from package blueprint-b47eadc99f3c525571d3834cc61b974453bc6be2
$ kpt alpha rpkg copy blueprint-b47eadc99f3c525571d3834cc61b974453bc6be2 --revision=v10 --namespace=default
```

<!--mdtogo-->