---
title: "`apply-time-mutation`"
linkTitle: "apply-time-mutation"
type: docs
description: >
  Specify one or more substitutions to make at apply time using dependencies as input.
---

The `config.kubernetes.io/apply-time-mutation` annotation specifies one or more
substitutions to make at apply time using dependencies as input.

### Schema

The annotation value accepts a list of substitutions, formatted as a YAML string.

- (\[\][Substitution]), required

    A list of one or more substitutions to perform on the annotated object.

#### Substitution

A substitution is a modification to a specific target field.

-   **sourceRef**: ([ObjectReference]), required

    Reference to the source resource, the dependency for this substitution.

-   **sourcePath**: (string), required

    The source resource field to read from as input, specified with [JSONPath].

-   **targetPath**: (string), required

    The target resource field to write to as output, specified with [JSONPath].

-   **token**: (string)

    The substring to replace in the target resource field.

    If the token is unspecified, the whole target field value will be replaced,
    allowing for replacement of non-string values using the source field type.

#### ObjectReference

A reference to a specific resource object.

-   **apiVersion**: (string)

    The group and version of the object resource.

    One of the following is required: apiVersion or group.

-   **group**: (string)

    The group of the object resource.

    Group is accepted as a version-less alternative to APIVersion.

    Group must be empty for resources in the "core" group.

    One of the following is required: apiVersion or group.

-   **kind**: (string), required

    The kind of the object resource.

-   **name**: (string), required

    The name of the object.

-   **namespace**: (string)

    The namespace of the object.

    Namespace is required for namespaced resources.

### Behavior

Like the `depends-on` feature, `apply-time-mutation` waits for dependencies 
(source resources) to be applied and reconciled before applying the resource
with the annotation.

Unlike the `depends-on` feature, `apply-time-mutation` modifies the annotated
resource before applying it.

#### Special cases

If the source resource is not part of the package being applied, the apply of
the target resource will fail with an error. 

The `apply-time-mutation` annotation is only enforced by `kpt live apply` and
`kpt live destroy`. Modifying or deleting these resources with other mechanisms
will not follow the rules specified by these annotations.

### JSONPath syntax

Since there is no formal specification for JSONPath, the supported syntax
depends on the chosen implimentation. In this case, kpt uses the
[ajson](https://github.com/spyzhov/ajson) library. For details about what
language features are supported, see the
[json-path-comparison table](https://cburgmer.github.io/json-path-comparison/).

### Example

In this example, pod-b depends on pod-a with two substitutions that replace
tokens in the same target field. The value of the SERVICE_HOST environment
variable of a container in pod-b will be updated to represent the host and port
from pod-a.

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
    - name: tcp
      containerPort: 80
---
apiVersion: v1
kind: Pod
metadata:
  name: pod-b
  namespace: test
  annotations:
    config.kubernetes.io/apply-time-mutation: |
      - sourceRef:
          kind: Pod
          name: pod-a
        sourcePath: $.status.podIP
        targetPath: $.spec.containers[?(@.name=="nginx")].env[?(@.name=="SERVICE_HOST")].value
        token: ${pod-a-ip}
      - sourceRef:
          kind: Pod
          name: pod-a
        sourcePath: $.spec.containers[?(@.name=="nginx")].ports[?(@.name=="tcp")].containerPort
        targetPath: $.spec.containers[?(@.name=="nginx")].env[?(@.name=="SERVICE_HOST")].value
        token: ${pod-a-port}
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    ports:
    - name: tcp
      containerPort: 80
    env:
    - name: SERVICE_HOST
      value: "${pod-a-ip}:${pod-a-port}"
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

To verify that the SERVICE_HOST was mutated correctly:

```shell
# Read the IP of pod-a
kubectl get pod pod-a -n test \
  -o jsonpath='{.status.podIP}'
```

```shell
# Read the SERVICE_HOST of pod-b
kubectl get pod pod-b -n test \
  -o jsonpath='{.spec.containers[?(@.name=="nginx")].env[?(@.name=="SERVICE_HOST")].value}'
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

[Substitution]: /reference/annotations/apply-time-mutation/#substitution
[ObjectReference]: /reference/annotations/apply-time-mutation/#objectreference
[JSONPath]: /reference/annotations/apply-time-mutation/#jsonpath-syntax
