---
title: "Sink"
linkTitle: "sink"
type: docs
description: >
   Specify a directory as an output sink package
---

<!--mdtogo:Short
    Specify a directory as an output sink package
-->

Implements a [sink function] by reading STDIN and writing configuration.

Sink will not prune / delete files for delete resources because it only knows
about files for which it sees input resources.

### Examples

<!--mdtogo:Examples-->

```sh
# run a function using explicit sources and sinks
kpt fn source DIR/ |
  kpt fn run --image gcr.io/example.com/my-fn |
  kpt fn sink DIR/
```

<!--mdtogo-->

### Synopsis

<!--mdtogo:Long-->

```sh
kpt fn sink [DIR]

DIR:
  Path to a package directory.  Defaults to stdout if unspecified.
```

<!--mdtogo-->

### Next Steps

- Learn about [functions concepts] like sources, sinks, and pipelines.
- See more examples of sink functions in the functions [catalog].

[sink function]: ../../../concepts/functions/#sink-function
[functions concepts]: ../../../concepts/functions/
[catalog]: ../../guides/consumer/function/sinks
