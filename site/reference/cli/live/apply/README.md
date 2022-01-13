---
title: "`apply`"
linkTitle: "apply"
type: docs
description: >
  Apply a package to the cluster (create, update, prune).
---

<!--mdtogo:Short
    Apply a package to the cluster (create, update, prune).
-->

`apply` creates, updates and deletes resources in the cluster to make the remote
cluster resources match the local package configuration.

### Synopsis

<!--mdtogo:Long-->

```
kpt live apply [PKG_PATH | -] [flags]
```

#### Args

```
PKG_PATH | -:
  Path to the local package which should be applied to the cluster. It must
  contain a Kptfile with inventory information. Defaults to the current working
  directory.
  Using '-' as the package path will cause kpt to read resources from stdin.
```

#### Flags

```
--dry-run:
  It true, kpt will validate the resources in the package and print which
  resources will be applied and which resources will be pruned, but no resources
  will be changed.
  If the --server-side flag is true, kpt will do a server-side dry-run, otherwise
  it will be a client-side dry-run. Note that the output will differ somewhat
  between the two alternatives.

--field-manager:
  Identifier for the **owner** of the fields being applied. Only usable
  when --server-side flag is specified. Default value is kubectl.

--force-conflicts:
  Force overwrite of field conflicts during apply due to different field
  managers. Only usable when --server-side flag is specified.
  Default value is false (error and failure when field managers conflict).

--install-resource-group:
  Install the ResourceGroup CRD into the cluster if it isn't already
  available. Default is false.

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

--poll-period:
  The frequency with which the cluster will be polled to determine
  the status of the applied resources. The default value is 2 seconds.

--prune-propagation-policy:
  The propagation policy that should be used when pruning resources. The
  default value here is 'Background'. The other options are 'Foreground' and 'Orphan'.

--prune-timeout:
  The threshold for how long to wait for all pruned resources to be
  deleted before giving up. If this flag is not set, kpt live apply will wait
  until interrupted. In most cases, it would also make sense to set the
  --prune-propagation-policy to Foreground when this flag is set.

--reconcile-timeout:
  The threshold for how long to wait for all resources to reconcile before
  giving up. If this flag is not set, kpt live apply will wait until
  interrupted.

--server-side:
  Perform the apply operation server-side rather than client-side.
  Default value is false (client-side).

--show-status-events:
  The output will include the details on the reconciliation status
  for all resources. Default is `false`.

  Does not apply for the `table` output format.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# apply resources in the current directory
$ kpt live apply
```

```shell
# apply resources in the my-dir directory and wait up until 15 minutes 
# for all the resources to be reconciled before pruning
$ kpt live apply --reconcile-timeout=15m my-dir
```

```shell
# apply resources and specify how often to poll the cluster for resource status
$ kpt live apply --reconcile-timeout=15m --poll-period=5s my-dir
```

<!--mdtogo-->
