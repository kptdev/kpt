---
title: "Tree"
linkTitle: "tree"
weight: 4
type: docs
description: >
  Render resources using a tree structure
---

<!--mdtogo:Short
    Render resources using a tree structure
-->

{{< asciinema key="pkg-tree" rows="10" preload="1" >}}

Tree displays the contents of a package using a tree structure to show
the relationships between directories, resources, and fields.

Tree supports a number of built-in fields such as replicas, images, ports,
etc. Additional fields may be printed by providing the `--field` flag

By default, kpt pkg tree uses Resource graph structure if any relationships
between resources (ownerReferences) are detected e.g. when printing
remote cluster resources rather than local package resources.
Otherwise, directory graph structure is used.

### Examples

{{% hide %}}

<!-- @makeWorkplace @verifyExamples-->

```
# Set up workspace for the test.
TEST_HOME=$(mktemp -d)
cd $TEST_HOME
```

<!-- @fetchPackage @verifyExamples-->

```shell
export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git
kpt pkg get $SRC_REPO/package-examples/helloworld-set@next my-dir
cd my-dir
```

{{% /hide %}}

<!--mdtogo:Examples-->
<!-- @pkgTree @verifyExamples-->

```shell
# print Resources using directory structure
kpt pkg tree
```

<!--mdtogo-->

### Synopsis

<!--mdtogo:Long-->

```
kpt pkg tree [DIR | -]
```

#### Args

```
DIR:
  Path to a package directory. Defaults to the current working directory.
```

#### Flags

```

```

<!--mdtogo-->
