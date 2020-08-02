---
title: "Init"
linkTitle: "init"
type: docs
description: >
   Initialize a package with a object to track previously applied resources
---
<!--mdtogo:Short
    Initialize a package with a object to track previously applied resources
-->

{{< asciinema key="live-init" rows="10" preload="1" >}}

The init command initializes a package with a template resource which will
be used to track previously applied resources so that they can be pruned
when they are deleted.

The template resource is required by other live commands
such as apply, preview and destroy.

### Examples
<!--mdtogo:Examples-->
```sh
# initialize a package
kpt live init my-dir/
```

```sh
# initialize a package with a specific name for the group of resources
kpt live init --inventory-namespace=test my-dir/
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt live init DIRECTORY [flags]
```

#### Args
```
DIR:
  Path to a package directory.  The directory must contain exactly
  one ConfigMap with the grouping object annotation.
```

#### Flags
```
--inventory-id:
  Identifier for group of applied resources. Must be composed of valid label characters.
--inventory-namespace:
  namespace for the inventory object.
```
<!--mdtogo-->
