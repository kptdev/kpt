---
title: "Init"
linkTitle: "init"
type: docs
description: >
   Initialize an empty package.
---
<!--mdtogo:Short
    Initialize an empty package.
-->

`init` initializes an existing empty directory as an empty kpt package.

### Synopsis
<!--mdtogo:Long-->
```
kpt pkg init [DIR] [flags]
```

#### Args

```
DIR:
  init fails if DIR does not already exist. Defaults to the current working directory.
```

#### Flags

```
--description
  Short description of the package. (default "sample description")

--name
  The package name. Defaults to the directory base name.

--keywords
  A list of keywords describing the package.

--site
  Link to page with information about the package.
```
<!--mdtogo-->

### Examples

{{% hide %}}

<!-- @makeWorkplace @verifyExamples-->
```
# Set up workspace for the test.
TEST_HOME=$(mktemp -d)
cd $TEST_HOME
```

{{% /hide %}}

<!--mdtogo:Examples-->

<!-- @pkgInit @verifyStaleExamples-->
```shell
# Creates a new Kptfile with metadata in the my-pkg directory.
mkdir my-pkg
kpt pkg init my-pkg --keyWords kpt.dev/app=cockroachdb \
    --description "my cockroachdb implementation"
```

```shell
# Creates a new Kptfile without metadata in the current directory.
kpt pkg init
```
<!--mdtogo-->
