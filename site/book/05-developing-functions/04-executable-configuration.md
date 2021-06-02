In this section we are going to take a look at two functions that follow the
[executable configuration pattern] we discussed at the beginning of this
chapter: `starlark` and `gatekeeper`.

## `starlark` function

[Starlark] is a python-like language designed for use in configuration files
that has several desirable properties: deterministic evaluation, hermetic
execution, and simplicity. The `starlark` function contains the interpreter for
the language, and accepts a `functionConfig` of kind `StarlarkRun` where you
provide your business logic.

The following is an example package with a `starlark` function declaration:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt-functions-catalog.git/examples/starlark/simple
```

It contains the following `functionConfig`:

```yaml
# simple/fn-config.yaml
apiVersion: fn.kpt.dev/v1alpha1
kind: StarlarkRun
metadata:
  name: set-namespace-to-prod
source: |
  # set the namespace on all resources
  def setnamespace(resources, namespace):
    for resource in resources:
      # mutate the resource
      resource["metadata"]["namespace"] = namespace
  setnamespace(ctx.resource_list["items"], "prod")
```

The `source` field includes the Starlark logic that sets the namespace on all
resources. Go ahead and render the package:

```shell
$ kpt fn render simple
```

You should now see that resources have `namespace` set to `prod`.

?> Refer to the [Functions Catalog](https://catalog.kpt.dev/starlark/v0.1/) for
details on how use this function.

## `gatekeeper` function

[Gatekeeper] project provides an extensible policy enforcement framework which
can be deployed as a Kubernetes admission controller (admission-time
enforcement) or as a kpt function (configuration authoring-time enforcement).
Policies are authored using custom resources that includes the business logic in
[Rego], a language designed for expressing policies over complex hierarchical
data structures.

The following is an example package with a `gatekeeper` function declaration:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt-functions-catalog.git/examples/enforce-gatekeeper/invalid-configmap
```

It contains the policy constraint containing the Rego logic which looks for
banned fields in `ConfigMap` which may contain credentials which you do not want
declared in a package and committed to Git:

```yaml
# invalid-configmap/resources.yaml (Excerpt)
apiVersion: templates.gatekeeper.sh/v1beta1
kind: ConstraintTemplate
metadata: # kpt-merge: /k8sbannedconfigmapkeysv1
  name: k8sbannedconfigmapkeysv1
spec:
  crd:
    spec:
      names:
        kind: K8sBannedConfigMapKeysV1
        validation:
          openAPIV3Schema:
            properties:
              keys:
                type: array
                items:
                  type: string
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |-
        package ban_keys

        violation[{"msg": sprintf("%v", [val])}] {
          keys = {key | input.review.object.data[key]}
          banned = {key | input.parameters.keys[_] = key}
          overlap = keys & banned
          count(overlap) > 0
          val := sprintf("The following banned keys are being used in the ConfigMap: %v", [overlap])
        }
---
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sBannedConfigMapKeysV1
metadata: # kpt-merge: /no-secrets-in-configmap
  name: no-secrets-in-configmap
spec:
  match:
    kinds:
      - apiGroups:
          - ""
        kinds:
          - ConfigMap
  parameters:
    keys:
      - private_key
```

If you render the package, you will get a validation error as expected:

```shell
$ kpt fn render invalid-configmap
```

?> Refer to the
[Functions Catalog](https://catalog.kpt.dev/enforce-gatekeeper/v0.1/) for
details on how use this function.

[executable configuration pattern]:
  /book/05-developing-functions/?id=executable-configuration
[starlark]: https://github.com/bazelbuild/starlark#starlark
[gatekeeper]: https://github.com/open-policy-agent/gatekeeper#gatekeeper
[rego]: https://www.openpolicyagent.org/docs/latest/#rego
