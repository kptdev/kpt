---
title: "Sink"
linkTitle: "sink"
type: docs
description: >
   Explicitly specify an output sink for a function
---
<!--mdtogo:Short
    Explicitly specify an output sink for a function
-->

Implements a sink by reading stdin and writing output to a local directory.

Sink will not prune / delete files for delete resources because it only knows
about files for which it sees input resources.

### Examples
<!--mdtogo:Examples-->
```sh
# run a function using explicit sources and sinks
kpt fn source DIR/ | kpt fn run --image gcr.io/example.com/my-fn | kpt fn sink DIR/
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt fn sink [DIR]

DIR:
  Path to a package directory.  Defaults to stdout if unspecified.
```
<!--mdtogo-->
