---
title: "Destroy"
linkTitle: "destroy"
type: docs
description: >
   Remove all previously applied resources in a package from the cluster
---
<!--mdtogo:Short
    Remove all previously applied resources in a package from the cluster
-->

{{< asciinema key="live-destroy" rows="10" preload="1" >}}

The destroy command removes all files belonging to a package from the cluster.

### Examples
<!--mdtogo:Examples-->
```sh
# remove all resources in a package from the cluster
kpt live destroy my-dir/
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt live destroy DIR
```

#### Args

```
DIR:
  Path to a package directory.  The directory must contain exactly
  one ConfigMap with the grouping object annotation.
```
<!--mdtogo-->
