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

{{< asciinema key="cfg-tree" rows="10" preload="1" >}}

Tree displays the contents of a package using a tree structure to show
the relationships between directories, resources, and fields.

Tree supports a number of built-in fields such as replicas, images, ports,
etc.  Additional fields may be printed by providing the `--field` flag

By default, kpt cfg tree uses Resource graph structure if any relationships
between resources (ownerReferences) are detected e.g. when printing
remote cluster resources rather than local package resources.
Otherwise, directory graph structure is used.

### Examples
<!--mdtogo:Examples-->
```sh
# print Resources using directory structure
kpt cfg tree my-dir/
```

```sh
# print replicas, container name, and container image and fields for Resources
kpt cfg tree my-dir --replicas --image --name
```

```sh
# print all common Resource fields
kpt cfg tree my-dir/ --all
```

```sh
# print the "foo"" annotation
kpt cfg tree my-dir/ --field "metadata.annotations.foo"
```

```sh
# print the status of resources with status.condition type of "Completed"
kubectl get all -o yaml | kpt cfg tree \
  --field="status.conditions[type=Completed].status"
```

```sh
# print live Resources from a cluster using owners for graph structure
kubectl get all -o yaml | kpt cfg tree --replicas --name --image
```

```sh
# print live Resources with status condition fields
kubectl get all -o yaml | kpt cfg tree \
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
kpt cfg tree [DIR] [flags]
```

#### Args

```
DIR:
  Path to a package directory.  Defaults to STDIN if not specified.
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
