---
title: "`local-config`"
linkTitle: "local-config"
type: docs
description: >
  Specify a resource to be skipped when applying.
---

The `config.kubernetes.io/local-config` annotation specifies a resource to be
skipped when applying.

These resources are generally used as input to kpt functions.

Because local resources are not applied, they don't need to follow a resource
schema known by the cluster. They just need to be a valid Kubernetes resource,
with apiVersion, kind, metadata, and name.

### Schema

The annotation value accepts string values of `true` and `false`.

Make sure to surround the value in quotes, otherwise it will be considered a
YAML boolean (invalid annotation), not a string.

### Behavior

Resources with the `local-config` annotation set to any value except `false`
will not be applied to the cluster when using `kpt live apply`.

### Example

In this example, the `ConfigMap` `cm-a` is local and not applied.

Create a new kpt package:

```shell
mkdir my-pkg
cd my-pkg
kpt pkg init
```

Configure a local `ConfigMap`:

```shell
cat > setters.yaml << EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: setters
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  key-a: value-a
  key-b: value-b
EOF
```

Create a namespace for your package:

```shell
kubectl create namespace test
```

Initialize the package inventory:

```shell
kpt live init
```

Apply the package to your Kubernetes cluster:

```shell
kpt live apply
```

If all goes well, the output should be empty.

To verify that the `ConfigMap` was not created:

```shell
kubectl get ConfigMap setters
```

The request should error:

```
Error from server (NotFound): configmaps "setters" not found
```
