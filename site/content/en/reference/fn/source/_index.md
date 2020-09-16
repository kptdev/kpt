---
title: "Source"
linkTitle: "source"
type: docs
description: >
   Specify a directory as an input source package
---

<!--mdtogo:Short
    Specify a directory as an input source package
-->

Implements a [source function] by reading configuration and writing to STDOUT.

### Examples

<!--mdtogo:Examples-->

```sh
# print to stdout configuration from DIR/ formatted as an input source
kpt fn source DIR/
```

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
kpt fn source [DIR...]

DIR:
  Path to a package directory.  Defaults to stdin if unspecified.
```

<!--mdtogo-->

### Next Steps

- Learn about [functions concepts] like sources, sinks, and pipelines.
- See more examples of source functions in the functions [catalog].

[source function]: ../../../concepts/functions/#source-function
[functions concepts]: ../../../concepts/functions/
[catalog]: ../../guides/consumer/function/sources
