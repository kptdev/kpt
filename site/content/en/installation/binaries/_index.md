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

```sh
# Download binaries using gsutil:
# Linux (x64)
gsutil cp gs://kpt-dev/latest/linux_amd64/kpt .
# macOS (x64)
gsutil cp gs://kpt-dev/latest/darwin_amd64/kpt .
# Windows (x64)
gsutil cp gs://kpt-dev/latest/windows_amd64/kpt.exe .

# For Linux/macOS
chmod +x kpt
sudo mv kpt /usr/local/bin/
```

**Note:** to run on **MacOS** the first time, it may be necessary to open the
program from the finder with *ctrl-click open*.

```sh
kpt version
```

[linux]: https://storage.googleapis.com/kpt-dev/latest/linux_amd64/kpt
[darwin]: https://storage.googleapis.com/kpt-dev/latest/darwin_amd64/kpt
[windows]: https://storage.googleapis.com/kpt-dev/latest/windows_amd64/kpt.exe
