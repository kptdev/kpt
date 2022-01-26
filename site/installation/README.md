# Installation

Users can get kpt in a variety of ways:

?> If you are migrating from kpt `v0.39`, please follow the [migration guide] to
kpt `v1.0+` binary.

## Binaries

Download pre-compiled binaries:

- [Linux (x64)][linux]
- [MacOS(x64)][darwin]

On Linux and MacOS, make it executable:

```shell
$ chmod +x kpt
```

?> On MacOS the first time, it may be necessary to open the
program from the finder with _ctrl-click open_.

Verify the version:

```shell
$ kpt version
```

## (Optional) enable shell auto-completion

kpt provides auto-completion support for several of the common shells.
To see the options for enabling shell auto-completion:
```shell
kpt completion -h
```

### Prerequisites
`kpt` depends on `bash-completion` in order to support auto-completion for the
bash shell. If you are using bash as your shell, you will need to install
`bash-completion` in order to use kpt's auto-completion feature.
`bash-completion` is provided by many package managers
(see [here][bash-completion]).

### Enable kpt auto-completion
The kpt completion script for a shell can be generated with the commands
`kpt completion bash`, `kpt completion zsh`, etc. Sourcing the completion script
in your shell enables auto-completion.

#### Enable auto-completion for your current shell
bash:
```shell
source <(kpt completion bash)
```
zsh:
```shell
source <(kpt completion zsh)
```
etc.
#### Enable kpt completion for all your shell sessions
bash:
```shell
echo 'source <(kpt completion bash)' >> ~/.bashrc
```
zsh:
```shell
echo 'source <(kpt completion zsh)' >> ~/.zshrc
```
etc.

<!-- gcloud and homebrew are not yet available for builds from the main branch.
## gcloud

Install with gcloud.

```shell
$ gcloud components install kpt
```

```shell
$ kpt version
```

The version of kpt installed using `gcloud` may not be the latest released version.

## Homebrew

Install the latest release with Homebrew on MacOS

```shell
$ brew tap GoogleContainerTools/kpt https://github.com/GoogleContainerTools/kpt.git
$ brew install kpt
```

```shell
$ kpt version
```
-->

## Docker

Use one of the kpt docker images.

| Feature   | `kpt` | `kpt-gcloud` |
| --------- | :---: | :----------: |
| kpt       |   ✓   |      ✓       |
| git       |   ✓   |      ✓       |
| diffutils |   ✓   |      ✓       |
| gcloud    |       |      ✓       |

### `kpt`

```shell
$ docker run gcr.io/kpt-dev/kpt:v1.0.0-beta.12 version
```

### `kpt-gcloud`

An image which includes kpt based upon the Google [cloud-sdk] alpine image.

```shell
$ docker run gcr.io/kpt-dev/kpt-gcloud:v1.0.0-beta.12 version
```

## Source

Install by compiling the source. This requires having Go version 1.16+:

```shell
$ go install -v github.com/GoogleContainerTools/kpt@main
```

kpt version will return `unknown` for binaries built from source:

```shell
$ kpt version
```

[gcr.io/kpt-dev/kpt]:
  https://console.cloud.google.com/gcr/images/kpt-dev/GLOBAL/kpt?gcrImageListsize=30
[gcr.io/kpt-dev/kpt-gcloud]:
  https://console.cloud.google.com/gcr/images/kpt-dev/GLOBAL/kpt-gcloud?gcrImageListsize=30
[cloud-sdk]: https://github.com/GoogleCloudPlatform/cloud-sdk-docker
[linux]:
  https://github.com/GoogleContainerTools/kpt/releases/download/v1.0.0-beta.12/kpt_linux_amd64
[darwin]:
  https://github.com/GoogleContainerTools/kpt/releases/download/v1.0.0-beta.12/kpt_darwin_amd64
[migration guide]: /installation/migration
[bash-completion]: https://github.com/scop/bash-completion#installation
