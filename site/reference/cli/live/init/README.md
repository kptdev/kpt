---
title: "`init`"
linkTitle: "init"
type: docs
description: >
  Initialize a package with the information needed for inventory tracking.
---

<!--mdtogo:Short
    Initialize a package with the information needed for inventory tracking.
-->

`init` initializes the package with the name, namespace and id of the resource
that will keep track of the package inventory.

### Synopsis

<!--mdtogo:Long-->

```
kpt live init [PKG_PATH] [flags]
```

#### Args

```
PKG_PATH:
  Path to the local package which should be updated with inventory information.
  It must contain a Kptfile. Defaults to the current working directory.
```

#### Flags

```
--force:
  Forces the inventory values to be updated, even if they are already set.
  Defaults to false.

--inventory-id:
  Inventory identifier for the package. This is used to detect overlap between
  packages that might use the same name and namespace for the inventory object.
  Defaults to an auto-generated value.

--name:
  The name for the ResourceGroup resource that contains the inventory
  for the package. Defaults to the name of the package.

--namespace:
  The namespace for the ResourceGroup resource that contains the inventory
  for the package. If not provided, kpt will check if all the resources
  in the package belong in the same namespace. If they do, that namespace will
  be used. If they do not, the namespace in the user's context will be chosen.

--rg-file:
  The name used for the file created for the ResourceGroup CR. Defaults to
  'resourcegroup.yaml'.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# initialize a package in the current directory.
$ kpt live init
```

```shell
# initialize a package with explicit namespace for the ResourceGroup.
$ kpt live init --namespace=test my-dir
```

<!--mdtogo-->
