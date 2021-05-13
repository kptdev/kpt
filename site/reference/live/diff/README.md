---
title: "`diff`"
linkTitle: "diff"
type: docs
description: >
   Display the diff between the local package and the live cluster resources.
---
<!--mdtogo:Short
    Display the diff between the local package and the live cluster resources.
-->

`diff` compares the live cluster state of each package resource against the 
local package config.

### Synopsis
<!--mdtogo:Long-->
```
kpt live diff [PKG_PATH | -]
```

#### Args
```
PKG_PATH | -:
  Path to the local package which should be diffed against the cluster. It must
  contain a Kptfile with inventory information. Defaults to the current working
  directory.
  Using '-' as the package path will cause kpt to read resources from stdin.
```

#### Environment Variables
```
KUBECTL_EXTERNAL_DIFF:
  Commandline diffing tool ('diff; by default) that will be used to show
  changes.
  
  # Use meld to show changes
  KPT_EXTERNAL_DIFF=meld kpt live diff
```

#### Exit statuses
```
The following exit values are returned:

  * 0: No differences were found.
  * 1: Differences were found.
  * >1 kpt live or diff failed with an error.
```
<!--mdtogo-->

### Examples
<!--mdtogo:Examples-->
```shell
# diff the config in the current directory against the live cluster resources.
kpt live diff

# specify the local diff program to use.
export KUBECTL_EXTERNAL_DIFF=meld; kpt live diff my-dir
```
<!--mdtogo-->
