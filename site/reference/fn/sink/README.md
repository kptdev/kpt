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
It is useful for chaining functions using Unix pipe. For more details, refer to
[Chaining functions].

### Synopsis

<!--mdtogo:Long-->

```shell
kpt fn sink DIR [flags]

DIR:
  Path to a local directory to resources to. Directory must exist.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# read resources from DIR directory, execute my-fn on them and write the
# output to DIR directory.
$ kpt fn source DIR |
  kpt fn run --image gcr.io/example.com/my-fn |
  kpt fn sink DIR
```

<!--mdtogo-->

[Chaining functions]: /book/04-using-functions/02-imperative-function-execution?id=chaining-functions-using-the-unix-pipe
[Function Specification]: /book/05-developing-functions/02-function-specification