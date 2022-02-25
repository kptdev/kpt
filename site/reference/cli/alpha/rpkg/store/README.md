---
title: "`store`"
linkTitle: "store"
type: docs
description: >
  Stores package resources into a remote package.
---

<!--mdtogo:Short
    Stores package resources into a remote package.
-->

`store` stores the resources provided either in `stdin` [Function Specification] wire format,
or in an optionally specified local directory and saves them to the remote package.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg store PACKAGE [DIR]
```

#### Args

```
PACKAGE:
  Name of the package where to store the resources.

DIR:
  Optional path to a local directory to read resources from.

```

#### Flags

```
--namespace
  Namespace containing the package.

```

<!--mdtogo-->

[function specification]:
  /book/05-developing-functions/01-functions-specification
