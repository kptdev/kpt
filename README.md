# KPT

Kubernetes Platform Toolkit

- Publish, Consume and Update packages of Kubernetes Resource Configuration.
- Develop and Update Configuration programmatically.
- Filter and Display Configuration packages.
- Apply Configuration to clusters.

`kpt` combines package management commands with upstream Kubernetes tools to provide a complete
toolchain for building platforms for Kubernetes Resources.

## Installation

Binaries:

darwin:

    wget https://storage.googleapis.com/kpt-dev/latest/darwin_amd64/kpt
    chmod +x kpt
    ./kpt version

linux:

    wget https://storage.googleapis.com/kpt-dev/latest/linux_amd64/kpt
    chmod +x kpt
    ./kpt version

windows:

    https://storage.googleapis.com/kpt-dev/latest/windows_amd64/kpt.exe

Source:

    GO111MODULE=on go get -v github.com/GoogleContainerTools/kpt

### [Documentation](googlecontainertools.github.io/kpt)

See the [docs](docs/README.md) for more information on how to use `kpt`.


---------------------
