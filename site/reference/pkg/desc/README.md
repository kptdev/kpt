---
title: "Desc"
linkTitle: "desc"
type: docs
description: >
   Display upstream package metadata
---
<!--mdtogo:Short
    Display upstream package metadata
-->

Desc displays information about the upstream package in tabular format.

### Examples

{{% hide %}}

<!-- @makeWorkplace @verifyExamples-->
```
# Set up workspace for the test.
TEST_HOME=$(mktemp -d)
cd $TEST_HOME
```

<!-- @fetchPackage @verifyExamples-->
```sh
export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git
kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.5.0 hello-world
```

{{% /hide %}}

<!--mdtogo:Examples-->

<!-- @pkgDesc @verifyExamples-->
```sh
# display description for the local hello-world package
kpt pkg desc hello-world/
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt pkg desc DIR

DIR:
  Path to a package directory
```
<!--mdtogo-->
