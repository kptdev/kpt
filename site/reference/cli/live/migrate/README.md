---
title: "`migrate`"
linkTitle: "migrate"
type: docs
description: >
Migrate a package and the inventory object to use the ResourceGroup CRD.
---
<!--mdtogo:Short
    Migrate a package and the inventory object to use the ResourceGroup CRD.
-->

`migrate` moves the inventory list, which contains the Group, Kind, Name and 
Namespace for every resource in the cluster that belongs to a package, into a 
`ResourceGroup` CR and moves the inventory information into the `Kptfile`.

Previous versions of `kpt` uses `ConfigMap` resources for storing the inventory
list and relies on an inventory template being among the applied resources. The
inventory template is just a regular `ConfigMap` manifest with a special
annotation that includes the information needed to look up any existing
inventory lists.

### Synopsis
<!--mdtogo:Long-->
```
kpt live migrate [PKG_PATH] [flags]
```

#### Args
```
PKG_PATH:
  Path to the local package. It must have a Kptfile and an existing inventory
  template in the root of the package. It defaults to the current directory.
```

#### Flags
```
--dry-run:
  Go through the steps of migration, but don't make any changes.

--force:
  Forces the inventory values in the Kptfile to be updated, even if they are
  already set. Defaults to false.

--name:
  The name for the ResourceGroup resource that contains the inventory
  for the package. Defaults to the same name as the existing ConfigMap
  inventory object.

--namespace:
  The namespace for the ResourceGroup resource that contains the inventory
  for the package. If not provided, it defaults to the same namespace as the
  existing ConfigMap inventory object.
```
<!--mdtogo-->

### Examples
<!--mdtogo:Examples-->

```shell
# Migrate the package in the current directory.
kpt live migrate
```
<!--mdtogo-->
