---
title: "Migrate"
linkTitle: "migrate"
type: docs
description: >
   Migrate the package inventory object to a ResourceGroup custom resource
---
<!--mdtogo:Short
    Migrate the package inventory object to a ResourceGroup custom resource
-->

**NOTE**: This command is not available unless the `RESOURCE_GROUP_INVENTORY`
environment variable is set.

The migrate command upgrades the inventory object from a
[**ConfigMap**](https://kubernetes.io/docs/concepts/configuration/configmap/)
to a **ResourceGroup** custom resource. The migrate performs the following steps:

1. Applies the **ResourceGroup** custom resource definition (see
   `kpt live install-resource-group`)
2. If a
   [**ConfigMap**](https://kubernetes.io/docs/concepts/configuration/configmap/)
   inventory object exists in the cluster, the inventory
   object is upgraded to a **ResourceGroup** custom resource (deleting the
   previous
   [**ConfigMap**](https://kubernetes.io/docs/concepts/configuration/configmap/)
   ).
3. If it has not already been created, the Kptfile inventory section
   is filled in. These values are used to create the **ResourceGroup**
   custom resource inventory object when the package is applied.
4. Deletes the local
   [**ConfigMap**](https://kubernetes.io/docs/concepts/configuration/configmap/)
   file (usually **inventory-template.yaml**).

### Examples
<!--mdtogo:Examples-->
```sh
# migrate from ConfigMap to ResourceGroup inventory object
export RESOURCE_GROUP_INVENTORY=1
kpt live migrate my-dir/
```

```sh
# check the steps that will occur for the migrate, but
# do not actually run them.
export RESOURCE_GROUP_INVENTORY=1
kpt live migrate my-dir/ --dry-run
```

```sh
# migrate from ConfigMap to ResourceGroup inventory object, forcing
# new values for the inventory object to be written to the Kptfile.
export RESOURCE_GROUP_INVENTORY=1
kpt live migrate my-dir/ --force
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt live migrate DIRECTORY [flags]
```

#### Args

```
DIR:
  Path to a package directory. The package must contain a Kptfile.
  If the package directory contains a ConfigMap inventory template
  file (usually named inventory-template.yaml), then this file
  will be deleted.
```

#### Flags

```
--dry-run:
  Do not actually run the migrate; only print out the steps that
  will occur.
--force:
  Set inventory values even if already set in Kptfile.
--name:
  Set the inventory object name, instead of default generated
  name (e.g. inventory-62308923). The user must make sure the
  inventory name does not collide with other inventory objects
  in the same namespace.
```
<!--mdtogo-->
