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

Implements reading STDIN and writing configuration.

Sink will not prune / delete files for delete resources because it only knows
about files for which it sees input resources.

### Synopsis

<!--mdtogo:Long-->

```shell
kpt fn sink [DIR] [flags]
```

#### Args

```
DIR:
  Path to a package directory. Defaults to stdout if unspecified.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# run a function using explicit sources and sinks
kpt fn source DIR/ |
  kpt fn eval --image gcr.io/example.com/my-fn |
  kpt fn sink DIR/
```

<!--mdtogo-->

### Next Steps

- Learn about [functions concepts] like sources, sinks, and pipelines.
- See more examples of sink functions in the functions [catalog].

[functions concepts]: /book/02-concepts/02-functions
[catalog]: https://catalog.kpt.dev/
