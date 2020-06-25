---
title: "Delete-setter"
linkTitle: "delete-setter"
weight: 4
type: docs
description: >
   Delete a setter for one field
---
<!--mdtogo:Short
    Delete a setter for one field
-->

{{< asciinema key="cfg-delete-setter" rows="10" preload="1" >}}

Setters provide a solution for template-free setting or substitution of field
values through package metadata (OpenAPI).  They are a safer alternative to
other substitution techniques which do not have the context of the
structured data -- e.g. using `sed` to replace values.

See the [deleting setters] guide for more info on creating setters.

### Examples
<!--mdtogo:Examples-->
```sh
# delete a setter replicas
kpt cfg delete-setter DIR/ replicas
```

<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt cfg delete-setter DIR NAME VALUE

DIR:
  Path to a package directory

NAME:
  The name of the setter to delete. e.g. replicas

```

<!--mdtogo-->

[deleting setters]: ../../../guides/producer/setters
