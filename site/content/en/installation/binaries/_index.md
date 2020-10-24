---
title: "binaries"
linkTitle: "binaries"
weight: 3
type: docs
description: >
    Download and run statically compiled go binaries.
---

Download pre-compiled binaries.

| Platform
| ------------------------
| [Linux (x64)][linux]
| [macOS (x64)][darwin]
| [Windows (x64)][windows]

For Linux/macOS, download binaries using `curl` alternatively:

```sh
# Linux (x64)
curl -Lo kpt https://storage.googleapis.com/kpt-dev/latest/linux_amd64/kpt

# macOS (x64)
curl -Lo kpt https://storage.googleapis.com/kpt-dev/latest/darwin_amd64/kpt
```

For Linux/macOS, make `kpt` executable and add it to your path:

```sh
chmod +x kpt && sudo mv kpt /usr/local/bin/
```

**Note:** to run on **MacOS** the first time, it may be necessary to open the
program from the finder with *ctrl-click open*.

```sh
kpt version
```

[linux]: https://storage.googleapis.com/kpt-dev/latest/linux_amd64/kpt
[darwin]: https://storage.googleapis.com/kpt-dev/latest/darwin_amd64/kpt
[windows]: https://storage.googleapis.com/kpt-dev/latest/windows_amd64/kpt.exe
