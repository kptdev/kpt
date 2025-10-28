---
title: "`source`"
linkTitle: "source"

description: |
  Source resources from a local directory
---

<!--mdtogo:Short
    Source resources from a local directory
-->

`source` reads resources from a local directory and writes them in
[Function Specification](/book/05-developing-functions/#functions-specification) wire format to `stdout`. The output of
the `source` can be pipe'd to commands such as `kpt fn eval` that accepts Function Specification wire format. `source`
is useful for chaining functions using Unix pipe. For more details, refer to
[Chaining functions](/book/04-using-functions/#chaining-functions-using-the-unix-pipe).

### Synopsis

<!--mdtogo:Long-->

```shell
kpt fn source [DIR] [flags]
```

#### Args

```shell
DIR:
  Path to the local directory containing resources. Defaults to the current
  working directory.
```

#### Flags

```shell
--fn-config:
  Path to the file containing `functionConfig`.

--include-meta-resources:
  (DEPRECATED) include-meta-resources is no longer necessary because meta
  resources are included by default with kpt version v1.0.0-beta.15+.

--output, o:
  If specified, the output resources are written to stdout in provided format.
  Allowed values:
  1. stdout(default): output resources are wrapped in ResourceList and written to stdout.
  2. unwrap: output resources are written to stdout, in multi-object yaml format.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# read resources from DIR directory and write the output on stdout.
$ kpt fn source DIR
```

```shell
# read resources from DIR directory, execute my-fn on them and write the
# output to DIR directory.
$ kpt fn source DIR |
  kpt fn eval - --image gcr.io/example.com/my-fn - |
  kpt fn sink DIR
```

<!--mdtogo-->
