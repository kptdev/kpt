---
title: "`cat`"
linkTitle: "cat"

description: |
  Print the contents of a package
---

<!--mdtogo:Short
    Print the contents of a package
-->

`cat` prints the contents of a package to stdout. KRM resources (YAML/JSON)
are formatted as YAML, while non-KRM text files are printed raw. Outputs are
separated by document separators.

### Synopsis

<!--mdtogo:Long-->

```shell
kpt pkg cat [FILE | DIR]
```

#### Args

```shell
FILE | DIR:
  Path to a file or a directory containing a kpt package. Displays all
  package files: KRM resources (YAML/JSON) are formatted by default,
  and non-KRM text files (e.g., README.md) are shown as raw content.
  Binary files are skipped. Defaults to the current directory.
```

<!--mdtogo-->

#### Flags

```shell
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
# Print all package contents from current directory.
$ kpt pkg cat
```

```shell
# Print a single resource file.
$ kpt pkg cat path/to/deployment.yaml
```

```shell
# Print a non-KRM file.
$ kpt pkg cat path/to/README.md
```

<!--mdtogo-->
