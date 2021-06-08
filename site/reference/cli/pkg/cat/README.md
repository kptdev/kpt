---
title: "`cat`"
linkTitle: "cat"
type: docs
description: >
  Print the resources in a file/directory
---

<!--mdtogo:Short
    Print the resources in a file/directory
-->

`cat` prints the resources in a file or directory as yaml to stdout.

### Synopsis

<!--mdtogo:Long-->

```
kpt pkg cat [FILE | DIR]
```

#### Args

```
FILE | DIR:
  Path to a directory either a directory containing files with KRM resources, or
  a file with KRM resource(s). Defaults to the current directory.
```

<!--mdtogo-->

#### Flags

```
--annotate
  Annotate resources with their file origins.

--format
  Format resource before printing. Defaults to true.

--recurse-subpackages, -R
  Print resources recursively in all the nested subpackages. Defaults to true.

--strip-comments
  Remove comments from yaml.

--style
  yaml styles to apply. May be 'TaggedStyle', 'DoubleQuotedStyle', 'LiteralStyle', 'FoldedStyle', 'FlowStyle'.
```

### Examples

<!--mdtogo:Examples-->

```shell
# Print resource from a file.
$ kpt pkg cat path/to/deployment.yaml
```

```shell
# Print resources from current directory.
$ kpt pkg cat
```

<!--mdtogo-->
