---
title: "Diff"
linkTitle: "diff"
type: docs
description: >
   Diff the local package config against the live cluster resources
---
<!--mdtogo:Short
    Diff the local package config against the live cluster resources
-->

The diff command compares the live cluster state of each pacakge
resource against the local package config.

### Examples
<!--mdtogo:Examples-->
```sh
# diff the config in "my-dir" against the live cluster resources
kpt live diff my-dir/

# specify the local diff program to use
export KUBECTL_EXTERNAL_DIFF=meld; kpt live diff my-dir/
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt live diff DIR

Output is always YAML.

KUBECTL_EXTERNAL_DIFF environment variable can be used to select your own diff command. By default, the "diff" command
available in your path will be run with "-u" (unicode) and "-N" (treat new files as empty) options.
```

#### Args

```
DIR:
  Path to a package directory.  The directory must contain exactly one ConfigMap with the inventory annotation.
```

#### Exit Status

```
The following exit values shall be returned:

0 No differences were found. 1 Differences were found. >1 kpt live or diff failed with an error.

Note: KUBECTL_EXTERNAL_DIFF, if used, is expected to follow that convention.
```
<!--mdtogo-->
