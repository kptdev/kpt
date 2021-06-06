---
title: "`tree`"
linkTitle: "tree"
weight: 4
type: docs
description: >
  Display resources, files and packages in a tree structure.
---

<!--mdtogo:Short
    Display resources, files and packages in a tree structure.
-->

`tree` displays resources, files and packages in a tree structure.

### Synopsis

<!--mdtogo:Long-->

```
kpt pkg tree [DIR]
```

<!--mdtogo-->

#### Args

```
DIR:
  Path to a directory containing KRM resource(s). Defaults to the current working directory.
```

### Examples

<!--mdtogo:Examples-->

{{% hide %}}

<!-- @makeWorkplace @verifyExamples-->

```
# Set up workspace for the test.
setupWorkspace
```
<!-- @pkgGet @verifyExamples-->
```shell
kpt pkg get https://github.com/kubernetes/examples.git/staging/cockroachdb@master 
cd cockroachdb
```

{{% /hide %}}

<!-- @pkgTree @verifyExamples-->
```shell
# Show resources in the current directory.
$ kpt pkg tree
```

<!--mdtogo-->
