---
title: "Fix"
linkTitle: "fix"
type: docs
description: >
   Fix a local package which is using deprecated features.
---
<!--mdtogo:Short
    Fix a local package which is using deprecated features.
-->

Fix reads the local package, modifies the package to use the latest kpt features
and fixes any deprecated feature traces.

### Examples

#### Example fix commands
<!--mdtogo:Examples-->
```sh
# print the fixes which will be made to the package without actually modifying
# resources
kpt pkg fix . --dry-run
```

```sh
# fix the package if it is using deprecated features
kpt pkg fix .
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt pkg fix LOCAL_PKG_DIRECTORY [flags]

Args:
  LOCAL_PKG_DIRECTORY:
    Local directory with kpt package. Directory must exist and
    contain a Kptfile.

Flags:
  --dry-run
    if set, the fix command shall only print the fixes which will be made to the
    package without actually fixing/modifying the resources.

```
<!--mdtogo-->
