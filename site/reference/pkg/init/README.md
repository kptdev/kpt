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

Init initializes an existing empty directory as an empty kpt package.

### Synopsis
<!--mdtogo:Long-->
```
kpt pkg init [DIR] [flags]
```

#### Args

```
DIR:
  Init fails if DIR does not already exist. Defaults to the current working directory.
```

#### Flags

```
--description
  short description of the package. (default "sample description")

--name
  package name.  defaults to the directory base name.

--keyWords
  list of keywords describing the package.

--site
  link to page with information about the package.
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
# writes Kptfile package meta if it doesn't already exists
mkdir my-pkg
kpt pkg init my-pkg --keyWords kpt.dev/app=cockroachdb \
    --description "my cockroachdb implementation"
```
<!--mdtogo-->
