# What is `thirdparty`?

This directory contains the files that are copied from 3rd-party projects and modified to fit kpt requirements.

# What is in `thirdparty`?

- `kyaml`: Files copied from [kyaml] v0.10.15 library
  - `runfn`: KRM function runner
- `cmdconfig`: Files copied from [cmd/config] v0.9.9 library
  - `commands`: Command files copied from [cmd/config]
- `cli-utils`: Files copied from [cli-utils] v0.25.0 library
  - `commands`: Command files copied from [cli-utils/cmd]

# Copyright and Licenses

All files in this directory will keep their original copyright notices at the beginning of the files.

All files in this directory will be under their original licenses. Licenses notices will be reserved.

# Contribute to Upstream

The modifications made in the 3rd-party files may be contributed to upstream. The contribution is determined case by case.

[kyaml]: https://github.com/kubernetes-sigs/kustomize/tree/8d72528eb5c73df80b20aae0a5e584c056879387/kyaml
[cmd/config]: https://github.com/kubernetes-sigs/kustomize/tree/b9c36caa1c5c6ee64926021841ea441773d0767c/cmd/config
[cli-utils]: https://github.com/kubernetes-sigs/cli-utils
[cli-utils/cmd]: https://github.com/kubernetes-sigs/cli-utils/tree/v0.25.0
