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
`ResourceGroup` CR and moves the inventory information into the a `ResourceGroup`
manifest on local disk.

0.39.x or earlier versions of `kpt` stored the inventory metadata in a
`ConfigMap` manifest in the package, and stored the inventory list in a
`ConfigMap` resource in the cluster. Running this command will move the
metadata into a `ResourceGroup` manifest in the `resourcegroup.yaml` file
and move the inventory list into a `ResourceGroup` CR.

Previous 1.0.0-alpha or 1.0.0-beta versions of kpt would store the inventory
metadata in the `Kptfile`. Running this command will move the metadata from
the `Kptfile` in a `ResourceGroup` manifest in the `resourcegroup.yaml` file.


### Synopsis

<!--mdtogo:Long-->

```
kpt live migrate [PKG_PATH] [flags]
```

#### Args

```
PKG_PATH:
  Path to the local package. It must have a Kptfile and inventory metadata
  in the package in either the ConfigMap, Kptfile or ResourceGroup format.
  It defaults to the current directory.
```

#### Flags

```
--dry-run:
  Go through the steps of migration, but don't make any changes.

--force:
  Forces the inventory values in the ResourceGroup manfiest to be updated,
  even if they are already set. Defaults to false.

--name:
  The name for the ResourceGroup resource that contains the inventory
  for the package. Defaults to the same name as the existing inventory
  object.

--namespace:
  The namespace for the ResourceGroup resource that contains the inventory
  for the package. If not provided, it defaults to the same namespace as the
  existing inventory object.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# Migrate the package in the current directory.
$ kpt live migrate
```

<!--mdtogo-->
