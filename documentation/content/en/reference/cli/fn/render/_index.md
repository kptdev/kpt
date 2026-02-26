---
title: "`render`"
linkTitle: "render"

description: |
  Render a package
---

<!--mdtogo:Short
   Render a package.
-->

`render` executes the pipeline of functions on resources in the package and
writes the output to the local filesystem in-place.

`render` executes the pipelines in the package hierarchy in a depth-first order.
For example, if a package A contains subpackage B, then the pipeline in B is
executed on resources in B and then the pipeline in A is executed on resources
in A and the output of the pipeline from package B. The output of the pipeline
from A is then written to the local filesystem in-place.

`render` formats the resources before writing them to the local filesystem.

If any of the functions in the pipeline fails, then the entire pipeline is
aborted and the local filesystem is left intact.

Refer to the [Declarative Functions Execution] for more details.

### Synopsis

<!--mdtogo:Long-->

```shell
kpt fn render [PKG_PATH] [flags]
```

#### Args

```shell
PKG_PATH:
  Local package path to render. Directory must exist and contain a Kptfile
  to be updated. Defaults to the current working directory.
```

#### Flags

```shell
--allow-exec:
  Allow executable binaries to run as function. Note that executable binaries
  can perform privileged operations on your system, so ensure that binaries
  referred in the pipeline are trusted and safe to execute.

--allow-network:
  Allow functions to access network during pipeline execution. Default: `false`. Note that this is applicable to container based functions only.

--image-pull-policy:
  If the image should be pulled before rendering the package(s). It can be set
  to one of always, ifNotPresent, never. If unspecified, always will be the
  default.

--output, o:
  If specified, the output resources are written to provided location,
  if not specified, resources are modified in-place.
  Allowed values: stdout|unwrap|<OUT_DIR_PATH>
  1. stdout: output resources are wrapped in ResourceList and written to stdout.
  2. unwrap: output resources are written to stdout, in multi-object yaml format.
  3. OUT_DIR_PATH: output resources are written to provided directory.
     The provided directory must not already exist.

--results-dir:
  Path to a directory to write structured results. Directory will be created if
  it doesn't exist. Structured results emitted by the functions are aggregated and saved
  to `results.yaml` file in the specified directory.
  If not specified, no result files are written to the local filesystem.
```

#### Kptfile Annotations

```shell
kpt.dev/save-on-render-failure:
  Controls whether partially rendered resources are saved when rendering fails.
  Set to "true" in the Kptfile metadata.annotations section to preserve the state
  of resources at the point of failure. This is useful for debugging render failures
  and understanding what changes were applied before the error occurred.
  This follows the same pattern as kpt.dev/bfs-rendering annotation.
  Default: false (failures will revert changes).
```

#### Environment Variables

```shell
KRM_FN_RUNTIME:
  The runtime to run kpt functions. It must be one of "docker", "podman" and "nerdctl".
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# Render the package in current directory
$ kpt fn render
```

```shell
# Render the package in current directory and save results in my-results-dir
$ kpt fn render --results-dir my-results-dir
```

```shell
# Render my-package-dir
$ kpt fn render my-package-dir
```

```shell
# Render the package in current directory and write output resources to another DIR
$ kpt fn render -o path/to/dir
```

```shell
# Render resources in current directory and write unwrapped resources to stdout
# which can be piped to kubectl apply
$ kpt fn render -o unwrap | kpt fn eval -i ghcr.io/kptdev/krm-functions-catalog/remove-local-config-resources:latest -o unwrap - | kubectl apply -f -
```

```shell
# Render resources in current directory, write the wrapped resources
# to stdout which are piped to 'set-annotations' function,
# the transformed resources are written to another directory
$ kpt fn render -o stdout \
| kpt fn eval - -i ghcr.io/kptdev/krm-functions-catalog/set-annotations:latest -o path/to/dir  -- foo=bar
```

```shell
# Render my-package-dir with podman as runtime for functions
$ KRM_FN_RUNTIME=podman kpt fn render my-package-dir
```

```shell
# Render my-package-dir with network access enabled for functions
$ kpt fn render --allow-network
```

```shell
# Example Kptfile with save-on-render-failure annotation
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-package
  annotations:
    kpt.dev/save-on-render-failure: "true"
...
```

<!--mdtogo-->

[declarative functions execution]:
  /book/04-using-functions/#declarative-function-execution
