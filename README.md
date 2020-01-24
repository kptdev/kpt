# KPT

kpt is a toolkit to help you manage, manipulate, customize, and apply Kubernetes Resource configuration data files.

- Fetch, update, and sync configuration files using git.
- Examine and modify configuration files.
- Generate, transform, validate configuration files using containerized functions.
- Apply configuration files to clusters.

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
