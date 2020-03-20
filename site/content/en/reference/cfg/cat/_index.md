---
title: "Cat"
linkTitle: "cat"
weight: 4
type: docs
description: >
   Print the resources in a package
---
<!--mdtogo:Short
    Print the resources in a package
-->

{{< asciinema key="cfg-count" rows="10" preload="1" >}}

Cat prints the resources in a package as yaml to stdout.

Cat is useful for printing only the resources in a package which might
contain other non-resource files.

### Examples
<!--mdtogo:Examples-->
```sh
# print Resource config from a directory
kpt cfg cat my-dir/
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
    kpt cfg cat DIR

    DIR:
      Path to a package directory
<!--mdtogo-->