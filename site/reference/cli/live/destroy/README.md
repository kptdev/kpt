---
title: "`destroy`"
linkTitle: "destroy"
type: docs
description: >
  Remove all previously applied resources in a package from the cluster
---

<!--mdtogo:Short
    Remove all previously applied resources in a package from the cluster
-->

`destroy` removes all files belonging to a package from the cluster.

### Synopsis

<!--mdtogo:Long-->

```
kpt live destroy [PKG_PATH | -]
```

#### Args

```
PKG_PATH | -:
  Path to the local package which should be deleted from the cluster. It must
  contain a Kptfile with inventory information. Defaults to the current working
  directory.
  Using '-' as the package path will cause kpt to read resources from stdin.
```

#### Flags

```
--dry-run:
  It true, kpt will print the resources that will be removed from the cluster,
  but no resources will be deleted.

--inventory-policy:
  Determines how to handle overlaps between the package being currently applied
  and existing resources in the cluster. The available options are:

    * strict: If any of the resources already exist in the cluster, but doesn't
      belong to the current package, it is considered an error.
    * adopt: If a resource already exist in the cluster, but belongs to a
      different package, it is considered an error. Resources that doesn't belong
      to other packages are adopted into the current package.

  The default value is `strict`.

--output:
  Determines the output format for the status information. Must be one of the following:

    * events: The output will be a list of the status events as they become available.
    * json: The output will be a list of the status events as they become available,
      each formatted as a json object.
    * table: The output will be presented as a table that will be updated inline
      as the status of resources become available.

  The default value is ‘events’.

--show-status-events:
  The output will include the details on the reconciliation status
  for all resources. Default is `false`.

  Does not apply for the `table` output format.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# remove all resources in the current package from the cluster.
$ kpt live destroy
```

<!--mdtogo-->
