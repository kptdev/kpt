---
title: "`tree`"
linkTitle: "tree"
weight: 4

description: |
  Display resources, files and packages in a tree structure.
---

<!--mdtogo:Short
    Display resources, files and packages in a tree structure.
-->

`tree` displays resources, files and packages in a tree structure.

### Synopsis

<!--mdtogo:Long-->

```shell
kpt pkg tree [DIR]
```

#### Args

```shell
DIR:
  Path to a local package directory. Defaults to the current directory.
  Displays KRM resources with their Kind and Name, and non-KRM text files
  as plain filenames. Dotfiles and symlinks are excluded.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# Show resources in the current directory.
$ kpt pkg tree
```

<!--mdtogo-->
