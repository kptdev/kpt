---
title: "Destroy"
linkTitle: "destroy"
type: docs
description: >
   Remove all previously applied resources in a package from the cluster
---
<!--mdtogo:Short
    Remove all previously applied resources in a package from the cluster
-->

{{< asciinema key="live-destroy" rows="10" preload="1" >}}

The destroy command removes all files belonging to a package from the cluster.

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
kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.5.0 my-dir
```

<!-- @createKindCluster @verifyExamples-->
```
kind delete cluster && kind create cluster
```

<!-- @initCluster @verifyExamples-->
```
kpt live init my-dir
kpt live apply my-dir
```

{{% /hide %}}

<!--mdtogo:Examples-->
<!-- @liveDestroy @verifyExamples-->
```sh
# remove all resources in a package from the cluster
kpt live destroy my-dir/
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt live destroy DIR
```

#### Args

```
DIR:
  Path to a package directory.  The directory must contain exactly
  one ConfigMap with the grouping object annotation.
```
<!--mdtogo-->
