---
title: "Installation"
linkTitle: "Installation"
weight: 20
type: docs
menu:
  main:
    weight: 1
---

Users can get kpt in a variety of ways:
1. Install with [GCloud](#GCloud)
1. Install with [Homebrew](#Homebrew) on MacOS
1. Use [Docker](#Docker)
1. Download [Binaries](#Binaries) 
1. Build from [Source](#Source)

## GCloud

Install with gcloud.

```shell
gcloud components install kpt
```

```shell
kpt version
```

The version of kpt installed using `gcloud` may not be the latest released version.

## Homebrew

Install the latest release with Homebrew on MacOS

```shell
brew tap GoogleContainerTools/kpt https://github.com/GoogleContainerTools/kpt.git
brew install kpt
```

```shell
kpt version
```

## Docker

Use one of the kpt docker images.

| Feature   | `kpt` | `kpt-gcloud` |
| --------- |:-----:|:------------:|
| kpt       | X     | X            |
| git       | X     | X            |
| diffutils | X     | X            |
| gcloud    |       | X            |

### [gcr.io/kpt-dev/kpt]

```shell
docker run gcr.io/kpt-dev/kpt version
```

### [gcr.io/kpt-dev/kpt-gcloud]

An image which includes kpt based upon the Google [cloud-sdk] alpine image.

```shell
docker run gcr.io/kpt-dev/kpt-gcloud version
```

[gcr.io/kpt-dev/kpt]: https://console.cloud.google.com/gcr/images/kpt-dev/GLOBAL/kpt?gcrImageListsize=30

[gcr.io/kpt-dev/kpt-gcloud]: https://console.cloud.google.com/gcr/images/kpt-dev/GLOBAL/kpt-gcloud?gcrImageListsize=30

[cloud-sdk]: https://github.com/GoogleCloudPlatform/cloud-sdk-docker

## Binaries

Download pre-compiled binaries.

| Platform
| ------------------------
| [Linux (x64)][linux]
| [macOS (x64)][darwin]
| [Windows (x64)][windows]

```shell
# For linux/mac
chmod +x kpt
```

**Note:** to run on **MacOS** the first time, it may be necessary to open the
program from the finder with *ctrl-click open*.

```shell
kpt version
```

## Source

Install by compiling the source.

```shell
GO111MODULE=on go get -v github.com/GoogleContainerTools/kpt
```

**Note:** `kpt version` will return *unknown* for binaries installed
with `go get`.

```shell
kpt help
```

[linux]: https://storage.googleapis.com/kpt-dev/latest/linux_amd64/kpt
[darwin]: https://storage.googleapis.com/kpt-dev/latest/darwin_amd64/kpt
[windows]: https://storage.googleapis.com/kpt-dev/latest/windows_amd64/kpt.exe
