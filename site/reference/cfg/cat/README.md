---
title: "Cat"
linkTitle: "cat"
weight: 4
type: docs
description: >
  Print the resources in a package
---

<!--mdtogo:Short
    Print the resources in a package
-->

{{< asciinema key="cfg-cat" rows="10" preload="1" >}}

Cat prints the resources in a package as yaml to stdout.

Cat is useful for printing only the resources in a package which might
contain other non-resource files.

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

{{% /hide %}}

<!--mdtogo:Examples-->

<!-- @cfgCat @verifyExamples-->
```sh
# print Resource config from a directory
kpt cfg cat my-dir/
```

<!--mdtogo-->

### Synopsis

<!--mdtogo:Long-->

```
kpt cfg cat DIR

DIR:
  Path to a package directory
```

<!--mdtogo-->

#### Flags

```sh
--annotate
  annotate resources with their file origins.

--dest string
  if specified, write output to a file rather than stdout

--exclude-non-local
  if true, exclude non-local-config in the output.

--format
  format resource config yaml before printing. (default true)

--function-config string
  path to function config to put in ResourceList -- only if wrapped in a ResourceList.

--include-local
  if true, include local-config in the output.

--recurse-subpackages, -R
  print resources recursively in all the nested subpackages. (default true)

--strip-comments
  remove comments from yaml.

--style
  yaml styles to apply.  may be 'TaggedStyle', 'DoubleQuotedStyle', 'LiteralStyle', 'FoldedStyle', 'FlowStyle'.

--wrap-kind string
  if set, wrap the output in this list type kind.

--wrap-version string
  if set, wrap the output in this list type apiVersion.
```
