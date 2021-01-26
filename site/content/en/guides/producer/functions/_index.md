---
title: "Functions"
linkTitle: "Functions"
weight: 4
type: docs
description: >
   Writing config functions to generate, transform, and validate resources.
---

## Functions Developer Guide

Config functions are conceptually similar to Kubernetes _controllers_ and
_validating webhooks_ -- they are programs which read resources as input, then
write resources as output (creating, modifying, deleting, or validating
resources).

Unlike controllers and validating webhooks, config functions can be run outside
of the Kubernetes control plane. This allows them to run in more contexts or
embedded in other systems. For example, functions could be:

- manually run locally
- automatically run locally as part of _make_, _mvn_, _go generate_, etc
- automatically run in CI/CD systems
- run by controllers as reconcile implementations

{{< svg src="images/fn" >}}

{{% pageinfo color="primary" %}}
Unlike pure-templating and DSL approaches, functions must be able to both
_read_ and _write_ resources, and specifically should be able to read resources
they have previously written -- updating the inputs rather generating new
resources.

This mirrors the level-triggered controller architecture used in the Kubernetes
control plane. A function reconciles the desired state of the resource
configuration to match the declared state specified by the functionConfig.

Functions that implement abstractions should update resources they have
generated in the past by reading them from the input.
{{% /pageinfo %}}

The following function runtimes are available in kpt:

| Runtime      | Read Resources From | Write Resources To  | Write Error Messages To | Validation Failure | Maturity |
| ------------ | ------------------- | ------------------- | ----------------------- | ------------------ | -------- |
| [Containers] | STDIN               | STDOUT              | STDERR                  | Exit Code          | Beta     |
| [Exec]       | STDIN               | STDOUT              | STDERR                  | Exit Code          | Alpha    |
| [Starlark]   | `ctx.resource_list` | `ctx.resource_list` | `log`                   | Exit Code          | Alpha    |

Additionally, the following libraries are available to create config functions:

| Library    | Link        |
| ---------- | ----------- |
| Golang     | [Go Fn Lib] |
| Typescript | [TS SDK]    |

## Input / Output

Functions read a `ResourceList`, modify it, and write it back out. The
`ResourceList` contains:

- (input+output) a list of resource `items`
- (input) configuration for the function
- (output) validation results

### ResourceList.items

Items are resources read from some source -- such as a package directory --
using a source function -- such as [`kpt fn source`] or [`helm-template`].

After a function adds, deletes or modifies items, the items will be written to
a sink directory using a sink function -- such as [`kpt fn sink`]. In most
cases the sink directory will be the same as the source directory.

```yaml
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  spec:
  ...
- apiVersion: v1
  kind: Service
  spec:
  ...
```

### ResourceList.functionConfig

Functions may optionally be configured using the `ResourceList.functionConfig`
field. `functionConfig` is analogous to a Deployment, and `items` is analogous
to the set of all resources in the Deployment controller in-memory cache (e.g.
all the resources in the cluster) -- this includes the ReplicaSets created,
updated and deleted for that Deployment.

```yaml
kind: ResourceList
functionConfig:
  apiVersion: example.com/v1alpha1
  kind: Foo
  spec:
    foo: bar
    ...
items:
  ...
```

{{% pageinfo color="primary" %}}
Some functions introduce a new resource type bespoke to the function instead of
using a ConfigMap as the functionConfig kind.
{{% /pageinfo %}}

### ResourceList.results

Functions may define validation results through the `results` field. When
functions are run using the `--results-dir`, each function's results field will
be written to a file under the specified directory.

```yaml
kind: ResourceList
functionConfig:
  apiVersion: example.com/v1alpha1
  kind: Foo
  spec:
    foo: bar
    ...
results:
- name: "kubeval"
  items:
  - severity: error # one of ["error", "warn", "info"] -- error code should be non-0 if there are 1 or more errors
    tags: # arbitrary metadata about the result
      error-type: "field"
    message: "Value exceeds the namespace quota, reduce the value to make the pod schedulable"
    resourceRef: # key to lookup the resource
      apiVersion: apps/v1
      kind: Deployment
      name: foo
      namespace: bar
    file:
      # optional if present as annotation
      path: deploy.yaml # read from annotation if present
      # optional if present as annotation
      index: 0 # read from annotation if present
    field:
      path: "spec.template.spec.containers[3].resources.limits.cpu"
      currentValue: "200" # number | string | boolean
      suggestedValue: "2" # number | string | boolean
  - severity: warn
    ...
- name: "something else"
  items:
  - severity: info
     ...
```

## Next Steps

- Learn how to [run functions].
- Find out how to structure a pipeline of functions from the
  [functions concepts] page.
- Consult the [fn command reference].

[Containers]: ./container
[Starlark]: ./starlark
[Exec]: ./exec
[Go Fn Lib]: ./golang/
[TS SDK]: ./ts/
[`kpt fn source`]: ../../../reference/fn/source/
[`helm-template`]: https://gcr.io/kpt-functions/helm-template/
[`kpt fn sink`]: ../../../reference/fn/sink/
[run functions]: ../../consumer/function/
[functions concepts]: ../../../concepts/functions/
[fn command reference]: ../../../reference/fn/
