---
title: "`status`"
linkTitle: "status"
type: docs
description: >
Display the status for the resources in the cluster
---

<!--mdtogo:Short
    Display shows the status for the resources in the cluster
-->

`status` shows the resource status for resources belonging to the package.

### Synopsis

<!--mdtogo:Long-->

```
kpt live status [PKG_PATH | -] [flags]
```

#### Args

```
PKG_PATH | -:
  Path to the local package for which the status of the package in the cluster
  should be displayed. It must contain a Kptfile with inventory information.
  Defaults to the current working directory.
  Using '-' as the package path will cause kpt to read resources from stdin.
```

#### Flags

```
--output:
  Determines the output format for the status information. Must be one of the following:

    * events: The output will be a list of the status events as they become available.
    * json: The output will be a list of the status events as they become available,
      each formatted as a json object.
    * table: The output will be presented as a table that will be updated inline
      as the status of resources become available.

  The default value is ‘events’.

--poll-period:
  The frequency with which the cluster will be polled to determine the status
  of the applied resources. The default value is 2 seconds.

--poll-until:
  When to stop polling for status and exist. Must be one of the following:

    * known: Exit when the status for all resources have been found.
    * current: Exit when the status for all resources have reached the Current status.
    * deleted: Exit when the status for all resources have reached the NotFound
      status, i.e. all the resources have been deleted from the live state.
    * forever: Keep polling for status until interrupted.

  The default value is ‘known’.

--timeout:
  Determines how long the command should run before exiting. This deadline will
  be enforced regardless of the value of the --poll-until flag. The default is
  to wait forever.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# Monitor status for the resources belonging to the package in the current
# directory. Wait until all resources have reconciled.
$ kpt live status
```

```shell
# Monitor status for the resources belonging to the package in the my-app
# directory. Output in table format:
$ kpt live status my-app --poll-until=forever --output=table
```

<!--mdtogo-->

[inventory template]: /reference/cli/live/apply/#prune
