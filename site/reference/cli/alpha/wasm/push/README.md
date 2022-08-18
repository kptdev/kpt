---
title: "`push`"
linkTitle: "push"
type: docs
description: >
Push WASM modules.
---

<!--mdtogo:Short
    Compress a WASM module and push it as an OCI image.
-->

`push` compresses a WASM module and push it as an OCI image.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha wasm push [LOCAL_PATH] [IMAGE]
```

#### Args

```
LOCAL_PATH:
  The path to the wasm file.
IMAGE:
  The desired name of an image. It must be a tag.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# compress ./my-fn.wasm and push it to gcr.io/my-org/my-fn:v1.0.0
$ kpt alpha wasm push ./my-fn.wasm gcr.io/my-org/my-fn:v1.0.0
```

<!--mdtogo-->
