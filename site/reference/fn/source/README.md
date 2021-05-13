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

Implements reading configuration and writing to STDOUT.

### Synopsis

<!--mdtogo:Long-->

```shell
kpt fn source [DIR] [flags]
```

#### Args

```
DIR:
  Path to a package directory. Defaults to stdin if unspecified.
```

#### Flags

```
--function-config:
  Path to the file containing `functionConfig` for the function. This `functionConfig`
  will be used in `functionConfig` field in the output resource.

--wrap-kind:
  A string which will be used as the `kind` of output resource. By default it's
  'ResourceList'.

--wrap-version:
  A string which will be used as the `apiVersion` of output resource. By default it's
  'config.kubernetes.io/v1alpha1'.
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

<!-- @fetchPackage @verifyExamples-->

```shell
export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git
kpt pkg get $SRC_REPO/package-examples/helloworld-set@next DIR/
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
  kpt fn eval --image gcr.io/example.com/my-fn |
  kpt fn sink DIR/
```

<!--mdtogo-->

### Next Steps

- Learn about [functions concepts] like sources, sinks, and pipelines.
- See more examples of source functions in the functions [catalog].

[functions concepts]: /book/02-concepts/02-functions
[catalog]: https://catalog.kpt.dev/
