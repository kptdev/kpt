---
title: "`wasm`"
linkTitle: "wasm"

weight: 4
description: >
    Manage WASM modules as OCI images.
menu:
  main:
    parent: "alpha"
---

<!--mdtogo:Short
    Manage WASM modules as OCI images.
-->

<!--mdtogo:Long-->
The `wasm` command group contains subcommands for managing WASM modules as OCI images.
<!--mdtogo-->

{{% alert title="Warning" color="warning" %}}
WASM functions are no longer supported. kpt used to be able to run KRM functions compiled to WASM, with
the `--allow-alpha-wasm` flag of `kpt fn eval` and `kpt fn render`. That alpha feature has been removed:
the runtimes it relied on are no longer maintained, and the feature saw little use. The flag is still
accepted but deprecated, and every attempt to run a function on the WASM runtime fails with an error:
with `kpt fn render` that is every function in the pipeline, with `kpt fn eval` it is a function given
with `--image` or a `.wasm` file given with `--exec`. Run your functions as container images or
executables instead. See [#4881](https://github.com/kptdev/kpt/issues/4881) for background.
{{% /alert %}}

{{< listdocs >}}
