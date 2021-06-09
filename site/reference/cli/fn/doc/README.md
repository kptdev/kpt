---
title: "Doc"
linkTitle: "doc"
type: docs
description: >
  Display the documentation for a function
---

<!--mdtogo:Short
    Display the documentation for a function
-->

{{% hide %}}

<!-- @makeWorkplace @verifyExamples-->
```
# Set up workspace for the test.
setupWorkspace
```

{{% /hide %}}

### Synopsis

<!--mdtogo:Long-->

`kpt fn doc` invokes the function container with `--help` flag.

```
kpt fn doc --image=IMAGE
```

--image is a required flag. If the function supports --help, it will print the
documentation to STDOUT. Otherwise, it will exit with non-zero exit code and
print the error message to STDERR.

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

<!--@verifyExamples-->
```shell
# diplay the documentation for image gcr.io/kpt-fn/set-namespace:v0.1.1
kpt fn doc --image gcr.io/kpt-fn/set-namespace:v0.1.1
```

<!--mdtogo-->
