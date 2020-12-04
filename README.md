<img src="https://storage.googleapis.com/kpt-dev/docs/logo.png" width="50" height="50" />

# KPT

kpt is a toolkit to help you manage, manipulate, customize, and apply Kubernetes Resource configuration data files.

- Fetch, update, and sync configuration files using git.
- Examine and modify configuration files.
- Generate, transform, validate configuration files using containerized functions.
- Apply configuration files to clusters.

## Installation

### Install with Gcloud

The version of kpt installed using `gcloud` may not be the latest released version.

```Shell
gcloud components install kpt
```

### Install with Homebrew

```Shell
brew tap GoogleContainerTools/kpt https://github.com/GoogleContainerTools/kpt.git
brew install kpt
```

### Download latest release

[Latest release][release]

### Install from source

```sh
GO111MODULE=on go get -v github.com/GoogleContainerTools/kpt
```

## Run using Docker image

[gcr.io/kpt-dev/kpt]

```sh
docker run gcr.io/kpt-dev/kpt version
```

### [Documentation][docs]

See the [docs] for more information on how to use `kpt`.

### [Roadmap][roadmap]

See the [roadmap] for more information about new features planned for `kpt`.

---

[linux]: https://storage.googleapis.com/kpt-dev/latest/linux_amd64/kpt
[darwin]: https://storage.googleapis.com/kpt-dev/latest/darwin_amd64/kpt
[windows]: https://storage.googleapis.com/kpt-dev/latest/windows_amd64/kpt.exe
[docs]: https://googlecontainertools.github.io/kpt
[release]: https://github.com/GoogleContainerTools/kpt/releases/latest
[gcr.io/kpt-dev/kpt]: https://console.cloud.google.com/gcr/images/kpt-dev/GLOBAL/kpt?gcrImageListsize=30
[roadmap]: https://github.com/GoogleContainerTools/kpt/blob/master/docs/ROADMAP.md
