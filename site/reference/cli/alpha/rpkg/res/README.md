---
title: "`res`"
linkTitle: "res"
type: docs
description: >
  Reads package resources.
---

<!--mdtogo:Short
    Reads package resources.
-->

`res` reads resources from a package and writes them in [Function Specification]
wire format to `stdout`. `res` can be used in place of `source` where the package
is in a registered package repository instead of on the local disk.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg res[ources] PACKAGE
```

#### Args

```
PACKAGE:
  Name of the package containing the resources.
```

#### Flags

```
--namespace
  Namespace containing the package.

```

<!--mdtogo-->

[function specification]:
  /book/05-developing-functions/01-functions-specification
