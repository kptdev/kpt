---
title: "Preview"
linkTitle: "preview"
type: docs
description: >
   Preview prints the changes apply would make to the cluster
---
<!--mdtogo:Short
    Preview prints the changes apply would make to the cluster
-->

{{< asciinema key="live-preview" rows="10" preload="1" >}}

The preview command will run through the same steps as apply, but
it will only print what would happen when running apply against the current
live cluster state. With the `--server-side` flag, the dry-run will
be performed on resources sent to the server (but not actually applied),
instead of less thorough dry-run calculations on the client.

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

<!-- @createKindCluster @verifyExamples-->
```
kind delete cluster && kind create cluster
```

<!-- @initCluster @verifyExamples-->
```
kpt live init my-dir
```

{{% /hide %}}

<!--mdtogo:Examples-->
<!-- @livePreview @verifyExamples-->
```sh
# preview apply for a package
kpt live preview my-dir/
```

<!-- @livePreview @verifyExamples-->
```sh
# preview destroy for a package
kpt live preview --destroy my-dir/
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt live preview DIRECTORY [flags]
```

#### Args

```
DIRECTORY:
  One directory that contain k8s manifests. The directory
  must contain exactly one ConfigMap with the grouping object annotation.
```

#### Flags

```
--destroy:
  If true, dry-run deletion of all resources.

--server-side:
  Boolean which performs the dry-run by sending the resource to the server.
  Default value is false (client-side dry-run). Available
  in version v0.36.0 and above. If not available, the user will see:
  "error: unknown flag".

--field-manager:
  String that can be set if --server-side flag is also set, which defines
  the resources field owner during dry-run. Available
  in version v0.36.0 and above. If not available, the user will see:
  "error: unknown flag".

--force-conflicts:
  Boolean that can be set if --server-side flag is also set, which overrides
  field ownership conflicts during dry-run. Available
  in version v0.36.0 and above. If not available, the user will see:
  "error: unknown flag".
```
<!--mdtogo-->
