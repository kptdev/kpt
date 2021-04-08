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
etc.  Additional fields may be printed by providing the `--field` flag

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
kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.5.0 my-dir
cd my-dir
```

{{% /hide %}}

<!--mdtogo:Examples-->
<!-- @pkgTree @verifyExamples-->
```shell
# print Resources using directory structure
kpt pkg tree
```

<!-- @pkgTree @verifyExamples-->
```shell
# print replicas, container name, and container image and fields for Resources
kpt pkg tree --replicas --image --name
```

<!-- @pkgTree @verifyExamples-->
```shell
# print all common Resource fields
kpt pkg tree --all
```

<!-- @pkgTree @verifyExamples-->
```shell
# print the "foo"" annotation
kpt pkg tree --field "metadata.annotations.foo"
```

<!-- @pkgTree @verifyStaleExamples-->
```shell
# print the status of resources piped from kubectl output with status.condition 
type of "Completed"
kubectl get all -o yaml | kpt pkg tree - \
  --field="status.conditions[type=Completed].status"
```

<!-- @pkgTree @verifyStaleExamples-->
```shell
# print live Resources from a cluster using owners for graph structure
kubectl get all -o yaml | kpt pkg tree --replicas --name --image
```

<!-- @pkgTree @verifyStaleExamples-->
```shell
# print live Resources with status condition fields
kubectl get all -o yaml | kpt pkg tree \
  --name --image --replicas \
  --field="status.conditions[type=Completed].status" \
  --field="status.conditions[type=Complete].status" \
  --field="status.conditions[type=Ready].status" \
  --field="status.conditions[type=ContainersReady].status"
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt pkg tree [DIR | -] [flags]
```

#### Args

```
DIR:
  Path to a package directory. Defaults to the current working directory.
```

#### Flags

```
--args:
  if true, print the container args field

--command:
  if true, print the container command field

--env:
  if true, print the container env field

--field:
  dot-separated path to a field to print

--image:
  if true, print the container image fields

--name:
  if true, print the container name fields

--ports:
  if true, print the container port fields

--replicas:
  if true, print the replica field

--resources:
  if true, print the resource reservations
```
<!--mdtogo-->
