---
title: "Init"
linkTitle: "init"
type: docs
description: >
   Initialize a package with a object to track previously applied resources
---
<!--mdtogo:Short
    Initialize a package with a object to track previously applied resources
-->

{{< asciinema key="live-init" rows="10" preload="1" >}}

The init command initializes a package with a template resource which will
be used to track previously applied resources so that they can be pruned
when they are deleted.

The template resource is required by other live commands
such as apply, preview and destroy.

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
{{% /hide %}}

<!--mdtogo:Examples-->

<!-- @liveInit @verifyExamples-->
```sh
# initialize a package
kpt live init my-dir/
```

{{% hide %}}

<!-- @removeInventoryTemplate @verifyExamples-->
```sh
rm my-dir/inventory-template.yaml
```

{{% /hide %}}

<!-- @liveInit @verifyExamples-->
```sh
# initialize a package with a specific name for the group of resources
kpt live init --namespace=test my-dir/
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt live init DIRECTORY [flags]
```

#### Args

```
DIR:
  Path to a package directory.  The directory must contain exactly
  one ConfigMap with the grouping object annotation.
```

#### Flags

```
--inventory-id:
  Identifier for group of applied resources. Must be composed of valid label characters.
--namespace:
  namespace for the inventory object. If not provided, kpt will check if all the resources
  in the package belong in the same namespace. If they are, that namespace will be used. If
  they are not, the namespace in the user's context will be chosen.
```
<!--mdtogo-->
