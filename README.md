# KPT

kpt is a toolkit to help you manage, manipulate, customize, and apply Kubernetes Resource configuration data files.

- Fetch, update, and sync configuration files using git.
- Examine and modify configuration files.
- Generate, transform, validate configuration files using containerized functions.
- Apply configuration files to clusters.

## Installation

Grab prebuilt Binaries:

| Platform                 
| ------------------------ 
| [Linux (x64)][linux]     
| [macOS (x64)][darwin]    
| [Windows (x64)][windows] 

or build from source:

    GO111MODULE=on go get -v github.com/GoogleContainerTools/kpt

then:

    # For linux/mac
    chmod +x kpt

    ./kpt version

### [Documentation](https://googlecontainertools.github.io/kpt)

See the [docs](https://googlecontainertools.github.io/kpt) for more information on how to use `kpt`.

---

[linux]: https://storage.googleapis.com/kpt-dev/latest/linux_amd64/kpt
[darwin]: https://storage.googleapis.com/kpt-dev/latest/darwin_amd64/kpt
[windows]: https://storage.googleapis.com/kpt-dev/latest/windows_amd64/kpt.exe
