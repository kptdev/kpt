---
title: "`pull`"
linkTitle: "pull"
type: docs
description: >
Pull WASM modules.
---

<!--mdtogo:Short
    Fetch and decompress OCI image to WASM module.
-->

`pull` fetches and decompressed OCI image to WASM module.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha wasm pull [IMAGE] [LOCAL_PATH]
```

#### Args

```
IMAGE:
  The name of an image. It can be either a tag or a digest.
LOCAL_PATH:
  The desired path for the wasm file. e.g. /tmp/my-fn.wasm
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# pull image gcr.io/my-org/my-fn:v1.0.0 and decompress it to ./my-fn.wasm
$ kpt alpha wasm pull gcr.io/my-org/my-fn:v1.0.0 ./my-fn.wasm
```

<!--mdtogo-->
