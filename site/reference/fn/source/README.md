---
title: "Source"
linkTitle: "source"
type: docs
description: >
   Specify a directory as an input source package
---

<!--mdtogo:Short
    Specify a directory as an input source package
-->

Implements a [source function] by reading configuration and writing to STDOUT.

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
kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.5.0 DIR/
```

{{% /hide %}}

<!--mdtogo:Examples-->

<!-- @fnSource @verifyExamples-->
```shell
# print to stdout configuration from DIR/ formatted as an input source
kpt fn source DIR/
```

```shell
# run a function using explicit sources and sinks
kpt fn source DIR/ |
  kpt fn run --image gcr.io/example.com/my-fn |
  kpt fn sink DIR/
```

<!--mdtogo-->

### Synopsis

<!--mdtogo:Long-->

```shell
kpt fn source [DIR...]

DIR:
  Path to a package directory.  Defaults to stdin if unspecified.
```

<!--mdtogo-->

### Next Steps

- Learn about [functions concepts] like sources, sinks, and pipelines.
- See more examples of source functions in the functions [catalog].

[source function]: /concepts/functions/#source-function
[functions concepts]: /concepts/functions/
[catalog]: /guides/consumer/function/catalog/sources/
