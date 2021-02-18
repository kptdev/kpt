---
title: "Init (alpha)"
linkTitle: "init (alpha)"
type: docs
description: >
   Initialize a package with a object to track previously applied resources
---
<!--mdtogo:Short
    Initialize a package with a object to track previously applied resources
-->

{{< asciinema key="live-init" rows="10" preload="1" >}}

**NOTE**: This alpha version of the command is for the new ResourceGroup as
inventory object functionality. The alpha version of this command is not
available unless the `RESOURCE_GROUP_INVENTORY` environment variable is set.

The init command will initialize a package using the next generation inventory
object (**ResourceGroup** custom resource). See commands [`migrate`](../migrate) and
[`install-resource-group`](../install-resource-group) for more information. A Kptfile
is required in the package directory.

The inventory object is required by other live commands
such as apply, preview and destroy.

### Examples
<!--mdtogo:Examples-->
```sh
# initialize a package with the next generation inventory metadata
export RESOURCE_GROUP_INVENTORY=1
kpt live init my-dir/
```

```sh
# initialize a package with a specific name for the group of resources
export RESOURCE_GROUP_INVENTORY=1
kpt live init --namespace=test-namespace my-dir/
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
  Path to a package directory. The package directory must contain a Kptfile.
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
