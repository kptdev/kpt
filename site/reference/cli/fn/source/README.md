---
title: "`source`"
linkTitle: "source"
type: docs
description: >
  Source resources from a local directory
---

<!--mdtogo:Short
    Source resources from a local directory
-->

`source` reads resources from a local directory and writes them in [Function
Specification] wire format to `stdout`. The output of the `source` can be pipe'd
to commands such as `kpt fn eval` that accepts Function Specification wire
format. `source` is useful for chaining functions using Unix pipe. For more
details, refer to [Chaining functions].

### Synopsis

<!--mdtogo:Long-->

```
kpt fn source [DIR] [flags]
```

#### Args

```
DIR:
  Path to the local directory containing resources. Defaults to the current
  working directory.
```

#### Flags

```
--fn-config:
  Path to the file containing `functionConfig`.

--include-meta-resources:
  If enabled, meta resources (i.e. `Kptfile` and `functionConfig`) are included
  in the output of the command. By default it is disabled.

```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```
# read resources from DIR directory and write the output on stdout.
$ kpt fn source DIR
```

```
# read resources from DIR directory, execute my-fn on them and write the
# output to DIR directory.
$ kpt fn source DIR |
  kpt fn eval - --image gcr.io/example.com/my-fn - |
  kpt fn sink DIR
```

<!--mdtogo-->

[chaining functions]: /book/04-using-functions/02-imperative-function-execution?id=chaining-functions-using-the-unix-pipe
[function specification]: /book/05-developing-functions/01-functions-specification
