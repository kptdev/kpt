---
title: "`depends-on`"
linkTitle: "depends-on"
type: docs
description: >
  Specify one or more resource dependencies.
---

The `config.kubernetes.io/depends-on` annotation specifies one or more resource
dependencies.

### Schema

The annotation value accepts a list of resource references, delimited by commas.

#### Resource reference

A resource reference is a string that uniquely identifies a resource.

It consists of the group, kind, name, and optionally the namespace, delimited by
forward slashes.

| Resource Scope | Format |
| -------------- | ------ |
| namespace-scoped | `<group>/namespaces/<namespace>/<kind>/<name>` |
| cluster-scoped   | `<group>/<kind>/<name>` |

For resources in the "core" group, the empty string is used instead
(for example: `/namespaces/test/Pod/pod-a`).

### Example

In this example, pod-b depends on pod-a.

Create a new kpt package:

```shell
mkdir my-pkg
cd my-pkg
kpt pkg init
```

Configure two pods, with one that depends on the other:

```shell
cat > pods.yaml << EOF
kind: Pod
apiVersion: v1
metadata:
  name: pod-a
  namespace: test
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    ports:
    - containerPort: 80
---
kind: Pod
apiVersion: v1
metadata:
  name: pod-b
  namespace: test
  annotations:
    config.kubernetes.io/depends-on: /namespaces/test/Pod/pod-a
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    ports:
    - containerPort: 80
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

If all goes well, the output should look like this:

```
pod/pod-a created
1 resource(s) applied. 1 created, 0 unchanged, 0 configured, 0 failed
pod/pod-b created
1 resource(s) applied. 1 created, 0 unchanged, 0 configured, 0 failed
```

Delete the package from your Kubernetes cluster:

```shell
kpt live destroy
```

If all goes well, the output should look like this:

```
pod/pod-b deleted
1 resource(s) deleted, 0 skipped
pod/pod-a deleted
1 resource(s) deleted, 0 skipped
```
