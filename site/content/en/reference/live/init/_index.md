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

Alternatively, if the RESOURCE_GROUP_INVENTORY environment variable is set,
the init command will initialize a package using the next generation inventory
object (**ResourceGroup** custom resource). See commands `migrate` and
`install-resource-group` for more information.

The inventory object is required by other live commands
such as apply, preview and destroy.

### Examples
<!--mdtogo:Examples-->
```sh
# initialize a package
kpt live init my-dir/
```

```sh
# initialize a package with a specific name for the group of resources
kpt live init --namespace=test my-dir/
```

```sh
# initialize a package with the next generation inventory metadata
export RESOURCE_GROUP_INVENTORY=1
kpt live init my-dir/
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
  one ConfigMap with the grouping object annotation. If the
  RESOURCE_GROUP_INVENTORY environment variable is set, the
  package must have a Kptfile.
```

#### Flags

```
--inventory-id:
  Identifier for group of applied resources. Must be composed of valid label characters.
--namespace:
  namespace for the inventory object. If not provided, kpt will check if all the resources
  in the package belong in the same namespace. If they are, that namespace will be used. If
  they are not, the namespace in the user's context will be chosen.
```
<!--mdtogo-->
