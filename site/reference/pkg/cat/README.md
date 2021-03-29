---
title: "Cat"
linkTitle: "cat"
type: docs
description: >
  Print the KRM resources in a file/directory
---

<!--mdtogo:Short
    Print the KRM resources in a file/directory
-->

Cat prints the resources in a package as yaml to stdout.

Cat is useful for printing the KRM resources in a file/directory which might
contain other non-resource files.

### Synopsis

<!--mdtogo:Long-->

```
kpt pkg cat [FILE | DIR]

DIR:
  Path to a directory with KRM resource files. Defaults to the current working directory.

FILE:
  Path to a KRM file.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```
# print Resource config from a file
kpt pkg cat path/to/deployment.yaml
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
```

```
# print Resource config from current directory
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
  format resource config yaml before printing. (default true)

--include-local
  if true, include local-config in the output.

--only-local
  if true, print only the local-config.

--recurse-subpackages, -R
  print resources recursively in all the nested subpackages. (default true)

--strip-comments
  remove comments from yaml.

--style
  yaml styles to apply. may be 'TaggedStyle', 'DoubleQuotedStyle', 'LiteralStyle', 'FoldedStyle', 'FlowStyle'.
```
