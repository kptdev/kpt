---
title: "Delete-setter"
linkTitle: "delete-setter"
weight: 4
type: docs
description: >
   Delete a setter
---
<!--mdtogo:Short
    Delete a setter
-->

{{< asciinema key="cfg-delete-setter" rows="10" preload="1" >}}

Setters provide a solution for template-free setting or substitution of field
values through package metadata (OpenAPI).  They are a safer alternative to
other substitution techniques which do not have the context of the
structured data -- e.g. using `sed` to replace values.

See the [setters] guide for more info on creating and deleting setters.

### Examples
<!--mdtogo:Examples-->
```sh
# delete a setter replicas
kpt cfg delete-setter DIR/ replicas
```

<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```sh
kpt cfg delete-setter DIR NAME

DIR:
  Path to a package directory

NAME:
  The name of the setter to delete. e.g. replicas

```

<!--mdtogo-->

```sh

--recurse-subpackages, -R
  Delete setter recursively in all the nested subpackages

```

[setters]: ../../../guides/producer/setters/#deleting-a-setter
