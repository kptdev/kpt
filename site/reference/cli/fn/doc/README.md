---
title: "Doc"
linkTitle: "doc"
type: docs
description: >
  Display the documentation for a function
---

<!--mdtogo:Short
    Display the documentation for a function
-->

### Synopsis

<!--mdtogo:Long-->

`kpt fn doc` invokes the function container with `--help` flag.
If the function supports `--help`, it will print the documentation to STDOUT.
Otherwise, it will exit with non-zero exit code and print the error message to STDERR.

```
kpt fn doc --image=IMAGE
```

#### Flags

```
--image, i: (required flag)
  Container image of the function e.g. `gcr.io/kpt-fn/set-namespace:v0.1`.
  For convenience, if full image path is not specified, `gcr.io/kpt-fn/` is added as default prefix.
  e.g. instead of passing `gcr.io/kpt-fn/set-namespace:v0.1` you can pass `set-namespace:v0.1`.
```

#### Environment Variables

```
KPT_FN_RUNTIME:
  The runtime to run kpt functions. It must be one of "docker", "podman" and "nerdctl".
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# display the documentation for image set-namespace:v0.1.1
kpt fn doc -i set-namespace:v0.1.1
```

<!--mdtogo-->
