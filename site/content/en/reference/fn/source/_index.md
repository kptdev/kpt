---
title: "Sink"
linkTitle: "sink"
type: docs
description: >
   Explicitly specify an output source for a function
---
<!--mdtogo:Short
    Explicitly specify an output source for a function
-->

Implements a Source by reading configuration and writing to command stdout.

### Examples
<!--mdtogo:Examples-->
```sh
# print to stdout configuration formatted as an input source
kpt fn source DIR/
```

```sh
# run a function using explicit sources and sinks
kpt fn source DIR/ | kpt run --image gcr.io/example.com/my-fn | kpt fn sink DIR/
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt fn source [DIR...]

DIR:
  Path to a package directory.  Defaults to stdin if unspecified.
```
<!--mdtogo-->
