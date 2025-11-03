---
title: "Chapter 7: Effective Customizations"
linkTitle: "Chapter 7: Effective Customizations"
description: |
    Kubernetes configuration packages and customizations go hand in hand; all the packaging tools enable package
    customization because almost every package will be adapted for each specific use. In this chapter we cover the effective 
    customizations techniques that kpt rendering and packaging enables. We show how providing customization through
    parameters has some
    [pitfalls](https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/declarative-application-management.md#parameterization-pitfalls).
    We recommend alternatives which do not hide the contents of packages behind facades. Some of these alternatives are only possible
    because kpt has made an investment into bulk editing with [KRM functions](../04-using-functions/) and upstream merging.
toc: true
menu:
  main:
    parent: "Book"
    weight: 70
---

## Prerequisites

Before reading this chapter you should familiarize yourself with [chapter 4](../04-using-functions/)
that talks about using functions as well as the [updating a package](../03-packages/#updating-a-package) section of 
[chapter 3](../03-packages/).

## Single Value Replacement

### Scenario

I have a single value replacement in my package. I don’t want to force package consumers 
to look through all the yaml files to find the occurrances the value I want them to set. It 
seems easier to just create a parameter for this value and have the user use the `Kptfile`
for setting the value.

Example storage bucket:

```yaml
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: my-bucket # kpt-set: ${project-id}-${name}
  namespace: ns-test # kpt-set: ${namespace}
  annotations:
    cnrm.cloud.google.com/force-destroy: "false"
    cnrm.cloud.google.com/project-id: my-project # kpt-set: ${project-id}
spec:
  storageClass: standard # kpt-set: ${storage-class}
  uniformBucketLevelAccess: true
  versioning:
    enabled: false
```

The corresponding Kptfile:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: bucket
info:
  description: A Google Cloud Storage bucket
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configMap:
        name: todo-bucket-name
        namespace: todo-namespace
        project-id: todo-project-id
        storage-class: standard
```


### Problems

1. As the package gets more popular, the single values inevitably expand to provide a facade to a large portion of the data.
   Instead of simplifying the `StorageBucket` configuration, we have made it more cmomplex. In this small example, almost
   half of the StorageBucket configuration is now covered with parameters.
1. Some values like resource names are used as references, so setting them in one place needs to trigger updates in all
   the places where they are referenced.
1. If additional resources that have similar values are added to the package new string replacements must be added
   and be marked up with project ID and namespace.
1. If a package is used as a sub-package the string replacement parameters must be surfaced to the parent package.
   If the parent package also expects some values to be set and the parameters do not exist in the sub-package, the sub-package
   must be updated with the parent package values.

### Solutions:

1. kpt allows the user to edit a particular value directly in the configuration data and will handle upstream merge.
   When [editing the yaml](../03-packages/#editing-a-package) directly, users are not confined to the parameters
   that the package author has provided. [kpt pkg update](../03-packages/#updating-a-package) merges the local edits
   made by consumer with the changes in the upstream package made by publisher. In the example above, `storageClass` can be set
   directly by the user.
1. Attributes like resource names which are often updated by consumers to add prefixes or suffixes
   (e.g. *-dev, *-stage, *-prod, na1-*, eu1-*) are best handled by the
   [ensure-name-substring](https://catalog.kpt.dev/function-catalog/ensure-name-substring/v0.2/) function that will handle dependency
   updates as well as capture all the resources in the package.
1. Instead of setting a particular value on a resource, a bulk operation can be applied to all the resources that fit a
   particular interface.  This can be done by a custom function or by the
   [set-namespace](https://catalog.kpt.dev/function-catalog/set-namespace/v0.4/),
   [search-and-replace](https://catalog.kpt.dev/function-catalog/search-replace/v0.2/),
   [set-labels](https://catalog.kpt.dev/function-catalog/set-labels/v0.2/) and
   [set-annotations](https://catalog.kpt.dev/function-catalog/set-annotations/v0.1/) functions.

The new bucket configuration:

```yaml
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: bucket
  annotations:
    cnrm.cloud.google.com/force-destroy: "false"
spec:
  storageClass: standard
  uniformBucketLevelAccess: true
  versioning:
    enabled: false
```

The customizations are now in the Kptfile:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: bucket
info:
  description: A Google Cloud Storage bucket
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:latest
      configMap:
        namespace: example-ns
    - image: ghcr.io/kptdev/krm-functions-catalog/ensure-name-substring:latest
      configMap:
        prepend: project111-
    - image: ghcr.io/kptdev/krm-functions-catalog/set-annotations:latest
      configMap:
        cnrm.cloud.google.com/project-id: project111
```

The mark up in the resource configuration YAML showing where the namespace value should
go is no longer needed.  The [set-namespace](https://catalog.kpt.dev/function-catalog/set-namespace/v0.4/) function is smart enough to 
find all the appropriate resources that need the namespace.

We have put in the starter name `bucket` and have an
[ensure-name-substring](https://catalog.kpt.dev/function-catalog/ensure-name-substring/v0.2/) 
that shows the package consumer that the project ID prefix is what we suggest.
However if they have a different naming convention they can alter the name 
prefix or suffix on all the resources in the package.

Since we are trying to set the annotation to the project ID we can run the 
[set-annotations](https://catalog.kpt.dev/function-catalog/set-annotations/v0.1/)
function once and the annotation is set on all the resources in the package. 
If we add additional resources or whole sub packages, we will get the consistent annotations across all resources 
without having to find all the places where annotations should go.

## Limiting Package Changes

### Scenario:

I’d like to limit what my package consumers can do with my package and it feels 
safer to just provide a string replacement in one place so they know not to 
alter the configuration outside of the few places that I designated as OK 
places to change.

Example deployment:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /nginx-deploy
  name: nginx-deploy
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: backend
          image: nginx:1.16.1 # kpt-set: nginx:${tag}
```

kpt configuration that uses a setter:
```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: dont-change-much
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:latest
      configMap:
        tag: 1.21
```

### Problems:

1. The limitation by parameters does not guarantee that consumers are in fact 
going to limit their changes to the parameters.  A popular pattern is using 
kustomize to change output of other tools no matter what parameters had.  In 
this particular case I am able to fork or patch this package and add:

```yaml
securityContext:
    runAsNonRoot: false
```

2. String replacements rarely describe the intent of the package author.
When additional resources are added I need additional places where parameters 
must be applied.  I can easily add other containers to this deployment and
the package author's rules are not clear and not easily validated.

### Solution:

1. General ways to describe policy already exist.  kpt has a
[gatekeeper](https://catalog.kpt.dev/function-catalog/gatekeeper/v0.2/)
function that allows the author to describe intended limitations for a class 
of resources or the entire package. This gives the consumer the freedom to customize 
and get an error or a warning when the policy is violated. 

In the sample provided by the function, we see how to provide a policy that will
clearly describe the intent using the [Rego policy language](https://www.openpolicyagent.org/docs/policy-language)
of the [Open Policy Agent (OPA)](https://www.openpolicyagent.org/):

The Kptfile uses the [gatekeeper](https://catalog.kpt.dev/function-catalog/gatekeeper/v0.2/) function to
ensure that resources comply with this policy every time `kpt fn render` is used.


### Example:

Create a kpt file with the following three yaml files:

1. `policy.yaml`

```yaml
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata: # kpt-merge: /disallowroot
  name: disallowroot
spec:
  crd:
    spec:
      names:
        kind: DisallowRoot
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |-
        package disallowroot
        violation[{"msg": msg}] {
          not input.review.object.spec.template.spec.securityContext.runAsNonRoot
          msg := "Containers must not run as root"
        }
---
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: DisallowRoot
metadata: # kpt-merge: /disallowroot
  name: disallowroot
spec:
  match:
    kinds:
      - apiGroups:
          - 'apps'
        kinds:
          - Deployment
```

2. `deployment-root-securitycontext.yaml.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /nginx-deploy
  name: nginx-deploy
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: backend
        image: nginx:1.16.1 # kpt-set: nginx:${tag}
      securityContext:
        runAsNonRoot: false
```

3. `Kptfile`

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: gatekeeper-disallow-root-user
pipeline:
  validators:
    - image: ghcr.io/kptdev/krm-functions-catalog/gatekeeper:latest
```

Now, run `kpt fn render` on the kpt package:

```bash
kpt fn render
Package "Gatekeeper": 
[RUNNING] "ghcr.io/kptdev/krm-functions-catalog/gatekeeper:latest"
[FAIL] "ghcr.io/kptdev/krm-functions-catalog/gatekeeper:latest" in 200ms
  Results:
    [error] apps/v1/Deployment/nginx-deploy: Containers must not run as root violatedConstraint: disallowroot
  Stderr:
    "[error] apps/v1/Deployment/nginx-deploy : Containers must not run as root"
    "violatedConstraint: disallowroot"
  Exit code: 1
```

The mutation pipeline fais because the Rego policy has been violated.

## Generation

### Scenario:

When using template languages I am able to provide conditional statements based 
on parameter values.  This allows me to ask the user for a little bit of 
information and generate a lot of boilerplate configuration.  Some template 
languages like [Jinja](https://palletsprojects.com/p/jinja/) are very robust and feature rich.

### Problems:

1. Increased usage and additional edge cases make a template a piece of code that requires testing and debugging.
1. The interplay between different conditionals and loops is interleaved in the template making it hard to understand
   what exactly is configuration and what is the logic that alters the configuration.  The consumer is left to resort to
   supplying different parameter values, executing the template rendering code and observing the results until a result that
   is correct for them emerges.
1. Templates are generally monolithic. When a change is introduced, the package consumers must either pay the cost of
   updating the templates or pay the cost of having to introduce more optional parameters.

### Solutions:

1. When the generated configuration is simple, consider just using a sub-package and running customizations using
   [single value replacement](#single-value-replacement) techniques.

