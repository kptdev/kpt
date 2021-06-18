---
title: "`sink`"
linkTitle: "sink"
type: docs
description: >
  Write resources to a local directory
---

<!--mdtogo:Short
   Write resources to a local directory
-->

`sink` reads resources from `stdin` and writes them to a local directory.
Resources must be in one of the following input formats:

1. Multi object YAML where resources are separated by `---`.

2. KRM Function Specification wire format where resources are wrapped in an
   object of kind ResourceList.

`sink` is useful for chaining functions using Unix pipe. For more details, refer
to [Chaining functions].

### Synopsis

<!--mdtogo:Long-->

```
kpt fn sink DIR [flags]

DIR:
  Path to a local directory to write resources to. The directory must not already exist.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# read resources from DIR directory, execute my-fn on them and write the
# output to DIR directory.
$ kpt fn source DIR |
  kpt fn eval - --image gcr.io/example.com/my-fn |
  kpt fn sink NEW_DIR
```

<!--mdtogo-->

[chaining functions]:
  /book/04-using-functions/02-imperative-function-execution?id=chaining-functions-using-the-unix-pipe
[function specification]:
  /book/05-developing-functions/01-functions-specification
