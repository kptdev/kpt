---
title: "Running a functions pipeline"
linkTitle: "Running a Functions Pipeline"
weight: 7
type: docs
description: >
    Compose functions into a pipeline.
---

## Composing a Pipeline

In order do something useful with a function, we need to compose a [Pipeline][concept-pipeline] with a
source and a sink function.

This guide covers how to use `kpt fn` to run a pipeline of functions. You can also use a container-based workflow orchestrator like [Cloud Build][cloud-build], [Tekton][tekton], or [Argo Workflows][argo].

### Example

First, initialize a git repo if necessary:

```sh
git init
```

Fetch an example configuraton package:

```sh
kpt pkg get https://github.com/GoogleContainerTools/kpt-functions-sdk.git/example-configs example-configs
cd example-configs
git add . && git commit -m 'fetched example-configs'
```

You can run a function, like [label-namespace], imperatively:

```sh
kpt fn run --image gcr.io/kpt-functions/label-namespace . -- label_name=color label_value=orange
```

You should see labels added to `Namespace` configuration files:

```sh
git status
```

Alternatively, you can run a function declaratively:

```sh
cat << EOF > kpt-func.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  annotations:
    config.k8s.io/function: |
      container:
        image:  gcr.io/kpt-functions/label-namespace
    config.kubernetes.io/local-config: "true"
data:
  label_name: color
  label_value: orange
EOF
```

You should see the same results as in the previous examples:

```sh
kpt fn run .
git status
```

You can have multiple function declarations in a directory. Let's add a second function:

```sh
cat << EOF > kpt-func2.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  annotations:
    config.k8s.io/function: |
      container:
        image:  gcr.io/kpt-functions/validate-rolebinding
    config.kubernetes.io/local-config: "true"
data:
  subject_name: bob@foo-corp.com
EOF
```

`fn run` executes both functions:

```sh
kpt fn run .
```

In this case, `validate-rolebinding` will find policy violations and fail with a non-zero exit code.

Refer to help pages for more details on how to use `kpt fn`

```sh
kpt fn run --help
```

## Next Steps

- Try running other functions in the [catalog].
- Get a quickstart on writing functions from the [function producer docs].
- Learn about [functions concepts] like sources, sinks, and pipelines.

[concept-pipeline]: ../../../../concepts/functions/#pipeline
[catalog]: ../catalog/
[label-namespace]: https://github.com/GoogleContainerTools/kpt-functions-sdk/blob/master/ts/hello-world/src/label_namespace.ts
[cloud-build]: https://cloud.google.com/cloud-build/
[tekton]: https://cloud.google.com/tekton/
[argo]: https://github.com/argoproj/argo
[function producer docs]: ../../../producer/functions/
[functions concepts]: ../../../../concepts/functions/
