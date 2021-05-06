---
title: "Cat"
linkTitle: "cat"
type: docs
description: >
  Print the resources in a file/directory
---

<!--mdtogo:Short
    Print the resources in a file/directory
-->

Prints the resources in a file/directory as yaml to stdout.

`cat` is useful for printing the resources in a file/directory which might
contain other non-resource files.

### Synopsis

<!--mdtogo:Long-->

```
kpt pkg cat [FILE | DIR]

DIR:
  Path to a directory with resources. Defaults to the current working directory.

FILE:
  Path to a resource file.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```
# print resource from a file
kpt pkg cat path/to/deployment.yaml
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
```

```
# print resources from current directory
kpt pkg cat
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
```

<!--mdtogo-->

#### Flags

```sh
--annotate
  annotate resources with their file origins.

--format
  format resource before printing. (default true)

--recurse-subpackages, -R
  print resources recursively in all the nested subpackages. (default true)

--strip-comments
  remove comments from yaml.

--style
  yaml styles to apply. may be 'TaggedStyle', 'DoubleQuotedStyle', 'LiteralStyle', 'FoldedStyle', 'FlowStyle'.
```
