---
title: "`init`"
linkTitle: "init"

description: |
  Initialize an empty package.
---

<!--mdtogo:Short
    Initialize an empty package.
-->

`init` initializes a directory as a kpt package by adding a Kptfile and a 
placeholder `README.md` file. If the directory does not exist, it will be created.

### Synopsis

<!--mdtogo:Long-->

```shell
kpt pkg init [DIR] [flags]
```

#### Args

```shell
DIR:
  init creates DIR if it does not already exist. Defaults to the current working directory.
```

#### Flags

```shell
--description
  Short description of the package. (default "sample description")

--keywords
  A list of keywords describing the package.

--site
  Link to page with information about the package.
```

<!--mdtogo-->

### Examples


<!--mdtogo:Examples-->

<!-- @pkgInit @verifyStaleExamples-->

```shell
# Creates a new Kptfile with metadata in the cockroachdb directory.
$ kpt pkg init cockroachdb --keywords "cockroachdb,nosql,db"  \
    --description "my cockroachdb implementation"
```

```shell
# Creates a new Kptfile without metadata in the current directory.
$ kpt pkg init
```

<!--mdtogo-->
