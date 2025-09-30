---
title: "Chapter 7: Effective Customizations"
linkTitle: "Chapter 7: Effective Customizations"
description: |
    Kubernetes configuration packages and customizations go hand in hand, all the packaging tools enable package
    customization, since every package needs to be adapted to each specific use. In this chapter we cover effective 
    customizations techniques that kpt rendering and packaging enables.  We show how providing customization through
    parameters has some
    [pitfalls](https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/declarative-application-management.md#parameterization-pitfalls)
    and recommend alternatives where the contents of the package are not hidden behind a
    facade. Some of these alternatives are only possible because kpt has made an investment into bulk editing with
    [KRM functions](../04-using-functions/) and upstream merging.
toc: true
menu:
  main:
    parent: "Book"
    weight: 70
---

## Prerequisites

Before reading this chapter you should familiarize yourself with [chapter 4](../04-using-functions/)
which talks about using functions as well as [updating a package page](../03-packages/#updating-a-package) in 
[chapter 3](../03-packages/).

## Single Value Replacement

### Scenario

I have a single value replacement in my package. I don’t want package consumers 
to look through all the yaml files to find the value I want them to set. It 
seems easier to just create a parameter for this value and have the user look 
at Kptfile for inputs.

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
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:v0.2.1
      configMap:
        name: todo-bucket-name
        namespace: todo-namespace
        project-id: todo-project-id
        storage-class: standard
```


### Problems

1. With package popularity the single values inevitably expand to provide a facade to a large portion of the data.
   That defeats the purpose of minimizing the cognitive load.  With this small example almost half of the StorageBucket
   configuration is now covered with parameters.
1. Some values like resource names are used as references so setting them in one place needs to trigger updates in all
   the places where they are referenced.
1. If additional resources that have similar values are added to the package new string replacements need to be added.
   In this case everything will need to also be marked up with project ID and namespace.
1. If a package is used as a sub-package the string replacement parameters need to be surfaced to the parent package and
   if the parent package already expects some values to be set and the parameters do not exist, the sub-package needs to 
   be updated.

### Solutions:

1. kpt allows the user to edit a particular value directly in the configuration data and will handle upstream merge.
   When [editing the yaml](../03-packages/#editing-a-package) directly the consumers are not confined to the parameters
   that the package author has provided. [kpt pkg update](../03-packages/#updating-a-package) merges the local edits
   made by consumer with the changes in the upstream package made by publisher. In this case `storageClass` can be set
   directly by the user.
1. Attributes like resource names which are often updated by consumers to add prefix or suffix
   (e.g. *-dev, *-stage, *-prod, na1-*, eu1-*) are best handled by the
   [ensure-name-substring](https://catalog.kpt.dev/ensure-name-substring/v0.1/) function that will handle dependency
   updates as well as capture all the resources in the package.
1. Instead of setting a particular value on a resource a bulk operation can be applied to all the resources that fit a
   particular interface.  This can be done by a custom function or by
   [set-namespace](https://catalog.kpt.dev/set-namespace/v0.2/),
   [search-and-replace](https://catalog.kpt.dev/search-replace/v0.2/),
   [set-labels](https://catalog.kpt.dev/set-labels/v0.1/) and
   [set-annotations](https://catalog.kpt.dev/set-annotations/v0.1/) functions.

New bucket configuration:

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

The suggested customizations are now in the Kptfile:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: bucket
info:
  description: A Google Cloud Storage bucket
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:v0.4.1
      configMap:
        namespace: example-ns
    - image: ghcr.io/kptdev/krm-functions-catalog/ensure-name-substring:v0.2.0
      configMap:
        prepend: project111-
    - image: ghcr.io/kptdev/krm-functions-catalog/set-annotations:v0.1.4
      configMap:
        cnrm.cloud.google.com/project-id: project111
```

The resource configuration YAML doesn't need to be marked up with where the 
namespace value needs to go.  The [set-namespace] function is smart enough to 
find all the appropriate resources that need the namespace.

We have put in the starter name `bucket` and have an [ensure-name-substring] 
that shows the package consumer that the project ID prefix is what we suggest.
However if they have a different naming convention they can alter the name 
prefix or suffix on all the resources in the package.

Since we are trying to set the annotation to the project ID we can use the 
[set-annotations] function one time and the annotation are going to be set on 
all the resources in the package.  If we add additional resources or whole 
sub packages we will get the consistent annotations across all resources 
without having to find all the places where annotations can go.

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
    - image: ghcr.io/kptdev/krm-functions-catalog/apply-setters:v0.2.1
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
need to be applied.  I can easily add other containers to this deployment and
the package author's rules are not clear and not easily validated.

### Solutions:

1. General ways to describe policy already exist.  kpt has a [gatekeeper] 
function that allows the author to describe intended limitations for a class 
of resources or the entire package giving the consumer the freedom to customize 
and get an error or a warning when the policy is violated. 

In the sample provided by the function we see how to provide a policy that will
clearly describe the intent using rego:

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

The Kptfile can enforce that resources comply with this policy every time
`kpt fn render` is used:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: gatekeeper-disallow-root-user
pipeline:
  validators:
    - image: ghcr.io/kptdev/krm-functions-catalog/gatekeeper:v0.2.1
```

## Generation

### Scenario:

When using template languages I am able to provide conditional statements based 
on parameter values.  This allows me to ask the user for a little bit of 
information and generate a lot of boilerplate configuration.  Some template 
languages like [Jinja](https://palletsprojects.com/p/jinja/) are very robust and feature rich.

### Problems:

1. Increased usage and additional edge cases make a template a piece of code that requires testing and debugging.
1. The interplay between different conditionals and loops is interleaved in the template making it hard to understand
   what exactly is configuration and what is the logic that alters the configuration.  The consumer is left with one
   choice supply different parameter values, execute the template rendering code and see what happens.
1. Templates are generally monolithic, when a change is introduced the package consumers need to either pay the cost of
   updating or the new consumers pay the cost of having to decipher more optional parameters.

### Solutions:

1. When the generated configuration is simple consider just using a sub-package and running customizations using
   [single value replacement](#single-value-replacement) techniques.
1. When a complex configuration needs to be generated the package author can create a generator function using turing
   complete languages and debugging tools. Example of such a function is
   [folder generation](https://catalog.kpt.dev/generate-folders/v0.1/). The output of the function is plain old KRM.
