# Installation

Users can get kpt CLI in a variety of ways:

## Binaries

Download pre-compiled binaries:

- [Linux (amd64)][linux-amd64]
- [Linux (arm64)][linux-arm64]
- [MacOS (amd64)][darwin-amd64]
- [MacOS (arm64)][darwin-arm64]

Optionally verify the [SLSA3 signatures](https://slsa.dev/) generated using the OpenSSF's
[slsa-framework/slsa-github-generator](https://github.com/slsa-framework/slsa-github-generator) during the release
process. To verify a release binary:

1. Install the verification tool from [slsa-framework/slsa-verifier#installation](https://github.com/slsa-framework/slsa-verifier#installation).
2. Download the signature file `multiple.intoto.jsonl` from the [GitHub releases page](https://github.com/kptdev/kpt/releases).
3. Run the verifier:

```shell
slsa-verifier verify-artifact --provenance-path multiple.intoto.jsonl --source-uri github.com/kptdev/kpt --source-versioned-tag <the-tag> kpt_<os>_<arch>
```

On Linux and MacOS, make it executable:

```shell
chmod +x kpt
```

On MacOS the first time, it may be necessary to open the program from the finder with _ctrl-click open_.

Verify the version:

```shell
kpt version
```

## (Optional) enable shell auto-completion

kpt provides auto-completion support for several of the common shells.
To see the options for enabling shell auto-completion:

```shell
kpt completion -h
```

### Prerequisites

Previous installations of kpt completion may have added the following line to the shell's config file
(e.g. `.bashrc`, `.zshrc`, etc.):

```shell
complete -C <KPT_PATH> kpt
```

This line needs to be removed for kpt's completion implementation to function
properly.

### Enable kpt auto-completion

The kpt completion script for a shell can be generated with the commands `kpt completion bash`, `kpt completion zsh`,
etc.
For instructions on how to enable the script for the given shell, see the help page with the commands
`kpt completion bash -h`, `kpt completion zsh -h`, etc.

## Homebrew

Install the latest release with Homebrew on MacOS.

```shell
brew tap kptdev/kpt https://github.com/kptdev/kpt.git
brew install kpt
```

```shell
kpt version
```

The version of kpt installed using `brew` can only be a tagged release, master releases are not shipped.

## Docker

Use one of the kpt docker images.

Running kpt via Docker does not install kpt on your machine. Each `docker run ...` invocation starts a temporary container, runs the command inside that container, prints output, and then exits.

| Feature   | `kpt` | `kpt-gcloud` |
| --------- | :---: | :----------: |
| kpt       |   ✓   |      ✓       |
| git       |   ✓   |      ✓       |
| diffutils |   ✓   |      ✓       |
| gcloud    |       |      ✓       |

### `kpt`

```shell
docker run ghcr.io/kptdev/kpt:{{< kpt_version >}} version
```

To use kpt with files on your host, mount your current directory into the container and set a working directory:

```shell
docker run --rm -v "$PWD":/workdir -w /workdir ghcr.io/kptdev/kpt:{{< kpt_version >}} pkg tree
docker run --rm -v "$PWD":/workdir -w /workdir ghcr.io/kptdev/kpt:{{< kpt_version >}} fn render
```

On Windows PowerShell, use `${PWD}.Path` for the current directory:

```shell
docker run --rm -v "${PWD}.Path:/workdir" -w /workdir ghcr.io/kptdev/kpt:{{< kpt_version >}} pkg tree
```

This pattern ensures kpt reads and writes files under `/workdir`, which maps to your current directory on the host.

### `kpt-gcloud`

An image which includes kpt based upon the Google [cloud-sdk] alpine image.

```shell
docker run ghcr.io/kptdev/kpt-gcloud:{{< kpt_version >}} version
```

Optionally, you can define a shell alias so Docker-based usage feels like a local CLI:

```shell
alias kpt='docker run --rm -v "$PWD":/workdir -w /workdir ghcr.io/kptdev/kpt:{{< kpt_version >}}'
```

On Windows PowerShell, you can define a function with a similar effect:

```shell
function kpt { docker run --rm -v "${PWD}.Path:/workdir" -w /workdir ghcr.io/kptdev/kpt:{{< kpt_version >}} $args }
```

After setting the alias, `kpt version` runs kpt in a container, and file changes persist in your current directory.

## Source

Install by compiling the source. This requires having Go version 1.21+:

```shell
go install -v github.com/kptdev/kpt@main
```

### Post-installation notes and troubleshooting

When installing with `go install`, the `kpt` binary is written to `$(go env GOPATH)/bin` by default. If `GOBIN` is set, Go installs binaries to that directory instead.

If that directory is not on your `PATH`, `kpt version` may return “command not found” even though the binary was installed successfully.

Verify where Go installed the binary, and confirm whether it is on your `PATH`:

```shell
go env GOPATH
ls "$(go env GOPATH)/bin"
echo "$PATH" | tr ':' '\n' | grep "$(go env GOPATH)/bin"
```

On Windows PowerShell:

```shell
go env GOPATH
Get-ChildItem "$(go env GOPATH)\bin" | Select-Object Name
$Env:Path -split ';' | Select-String -SimpleMatch "$(go env GOPATH)\bin"
```

If needed, add `$(go env GOPATH)/bin` to your `PATH` (adjust the shell profile file for your environment):

```shell
export PATH="$PATH:$(go env GOPATH)/bin"
```

On Windows PowerShell, you can update your user-level `PATH` for future sessions:

```shell
$gopath = (go env GOPATH)
[Environment]::SetEnvironmentVariable('Path', $Env:Path + ';' + "$gopath\bin", 'User')
```

Also note that `go install` may produce little or no output, and may complete very quickly, if Go reuses cached build artifacts.

If you suspect a stale module cache is affecting the build (for example, unexpected build results or missing updates), `go clean -modcache` can be a useful diagnostic step. This clears the module download cache and forces Go to re-fetch modules on the next build; it is not required for routine installs.

kpt version will return `unknown` for binaries built from source:

```shell
kpt version
```

[ghcr.io/kptdev/kpt]: https://github.com/kptdev/kpt/pkgs/container/kpt
[cloud-sdk]: https://github.com/GoogleCloudPlatform/cloud-sdk-docker

[linux-amd64]:
https://github.com/kptdev/kpt/releases/download/{{< kpt_version >}}/kpt_linux_amd64
[linux-arm64]:
https://github.com/kptdev/kpt/releases/download/{{< kpt_version >}}/kpt_linux_arm64
[darwin-amd64]:
https://github.com/kptdev/kpt/releases/download/{{< kpt_version >}}/kpt_darwin_amd64
[darwin-arm64]:
https://github.com/kptdev/kpt/releases/download/{{< kpt_version >}}/kpt_darwin_arm64
[bash-completion]: https://github.com/scop/bash-completion#installation
