---
title: "Init"
linkTitle: "init"
type: docs
description: >
   Initialize an empty package
---
<!--mdtogo:Short
    Initialize an empty package
-->

{{< asciinema key="pkg-init" rows="10" preload="1" >}}

Init initializes an existing empty directory as an empty kpt package.

**init is optional**: Any directory containing Kubernetes Resource
Configuration may be treated as remote package without the existence of
additional packaging metadata.

* Resource Configuration may be placed anywhere under DIR as *.yaml files.
* DIR may contain additional non-Resource Configuration files.
* DIR must be pushed to a git repo or repo subdirectory.

Init will augment an existing local directory with packaging metadata to help
with discovery.

Init will:

* Create a Kptfile with package name and metadata if it doesn't exist
* Create a README.md for package documentation if it doesn't exist.

### Examples
<!--mdtogo:Examples-->
```sh
# writes Kptfile package meta if not found
mkdir my-pkg
kpt pkg init my-pkg --tag kpt.dev/app=cockroachdb \
    --description "my cockroachdb implementation"
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt pkg init DIR [flags]
```

#### Args

```
DIR:
  Init fails if DIR does not already exist
```

#### Flags

```
--description
  short description of the package. (default "sample description")

--name
  package name.  defaults to the directory base name.

--tag
  list of tags for the package.

--url
  link to page with information about the package.
```
<!--mdtogo-->
