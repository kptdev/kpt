---
title: "source code"
linkTitle: "source code"
weight: 5
type: docs
description: >
    Dust off your go compiler and install from source.
---

Install by compiling the source.

```sh
GO111MODULE=on go get -v github.com/GoogleContainerTools/kpt
```

**Note:** `kpt version` will return *unknown* for binaries installed
with `go get`.

```sh
kpt help
```
