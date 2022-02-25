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

`res` reads resources from a package and writes them into a directory (if specified), or
into a [Function Specification] wire format to `stdout`.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg res[ources] PACKAGE [DIR]
```

#### Args

```
PACKAGE:
  Name of the package containing the resources.

DIR:
  Optional path to a local directory to write resources to. The directory must not already exist.

```

#### Flags

```
--namespace
  Namespace containing the package.

```

<!--mdtogo-->

[function specification]:
  /book/05-developing-functions/01-functions-specification
