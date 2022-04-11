# Title

* Author(s): Natasha Sarkar, natasha41575
* Approver: Sunil Arora, droot

## Why

`kpt fn render` and `kpt fn eval` now include meta resources (functionConfigs and the Kptfile), 
a breaking change from previous behavior. Users who use functions such as set-labels and set-namespace
will now see their KRM functions acting on resources that were previously excluded by default. 
For these users, we need to provide a way to preserve old behavior by introducing a mechanism to 
exclude certain resources, such as resources of a certain GVK or local configuration resources, from 
being acted on by kpt functions during `kpt fn render` and `kpt fn eval`.

### Background

Open source issue: https://github.com/GoogleContainerTools/kpt/issues/2930

#### Redefining and inclusion of meta resources

Previously, `kpt fn render` and `kpt fn eval` excluded functionConfigs and the Kptfile. A user could include t
hese "meta resources'' by setting the `--include-meta-resources` flag. There is a separate discussion 
and PR (https://github.com/GoogleContainerTools/kpt/pull/2894) with the following changes:

- The definition of meta-resources. Kpt no longer considers functionConfigs as meta resources; they are KRM 
resources like any other, and the fact that they are used to configure a KRM function does not mean they should be 
treated specially. With this change, "meta-resources" only includes the Kptfile.
- The `--include-meta-resources` flag is going away, and meta resources are processed by all kpt functions by default. 
With this change, there is no mechanism for users to exclude the Kptfile.

#### `kpt fn` selectors

kpt fn render and kpt fn eval currently support selector-based mechanisms to target certain resources.

Imperatively:

```shell
$ kpt fn eval [PKG_DIR] -i set-labels:v0.1.5 --match-kind Deployment
```

Declaratively:

```yaml
# pipeline of Kptfile
pipeline:
  mutators:
  - image: set-labels:v0.1.5
    selectors:
    - kind: Deployment
```

These selectors only allow you to list out which resources you want to include. There is no current mechanism to exclude a certain kind of resource. 
For example, the user may want to exclude the Kptfile.

#### Prior art in kustomize
The kustomize `replacements` feature (used in our new apply-replacements function), allows a mechanism to exclude certain resources from being modified. 
`replacements` is one of the most popular among kustomize users.

Kustomize has also had a few requests (such as this one: https://github.com/kubernetes/enhancements/pull/1232) to allow transformers to skip resources of a 
certain GVK, which gives us confidence that a similar feature in kpt would be welcome.

## Design

### A new `exclude` option inline with selectors

We can extend the current selector mechanism to allow exclusions. This can be achieved by new exclude flags for imperative workflow and a new exclude field in the 
Kptfile pipeline for declarative workflow. These correspond to the imperative match flags and declarative selectors field.

A few reasons in favor of this design:
- Flexible and powerful
- Allows users very explicit control of which resources are included at a per-function level
- It is consistent with our current selector mechanism.
- It is consistent with the syntax of kustomize replacements (used in our new apply-replacements function), which is popular and heavily used among the kubernetes community
- It is consistent with a highly requested feature in kustomize to allow transformers to exclude resources of a certain GVK.

#### Example: Exclude all resources of kind "Deployment"

```shell
$ kpt fn eval [PKG_DIR] -i set-labels:v0.1.5 --exclude-kind Deployment
```

```yaml
# pipeline of Kptfile
pipeline:
  mutators:
  - image: set-labels:v0.1.5
    exclude:
    - kind: Deployment
```

#### Example: Exclude all resources that have both group "apps" and kind "Deployment"

```shell
$ kpt fn eval [PKG_DIR] -i set-labels:v0.1.5 --exclude-kind Deployment --exclude-group apps
```
```yaml
# pipeline of Kptfile
pipeline:
  mutators:
  - image: set-labels:v0.1.5
    exclude:
    - kind: Deployment
      group: apps
```

#### Example: Exclude all resources that have either group "apps" or kind "Deployment"

```yaml
# pipeline of Kptfile
pipeline:
  mutators:
  - image: set-labels:v0.1.5
    exclude:
    - kind: Deployment
    - group: apps
```

With the current proposal, this is not possible imperatively with kpt fn eval.

#### Example: Select all resources that have group "apps", but do NOT have kind "Deployment"

```shell
$ kpt fn eval [PKG_DIR] -i set-labels:v0.1.5 --match-group apps --exclude-kind Deployment
```
```yaml
# pipeline of Kptfile
pipeline:
  mutators:
  - image: set-labels:v0.1.5
    selectors:
      group: apps
    exclude:
    - kind: Deployment
``` 

### Selecting by labels or annotations

We will support selection and exclusion with annotation and label selectors.

```shell
$ kpt fn eval [PKG_DIR] -i set-labels:v0.1.5 --match-annotation foo=bar --exclude-annotation config.kubernetes.io/local-config=true
```

```yaml
# pipeline of Kptfile
pipeline:
  mutators:
  - image: set-labels:v0.1.5
    selectors:
    - annotations:
        foo: bar
    exclude:
    - annotations:
        config.kubernetes.io/local-config: "true"
```

## Open Issues/Questions

### How should kpt functions handle meta resources?

One concern about this approach is that in order to exclude the Kptfile from being modified by horizontal
transformations like set-namespace and set-labels, the user will have to write an exclusion field for each function.
Arguably, the user shouldn't have to provide any special syntax for the Kptfile at all, because kpt and 
kpt functions should be able to handle it correctly. A mitigation for this is that our own horizontal kpt functions
can exclude the Kptfile within the function logic itself, and there will be no special handling logic in kpt. 

## Alternatives Considered

### A new annotation to decide if a kpt function should modify it

We can introduce a new annotation that determines if functions should modify it.

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example
  annotations:
    config.kubernetes.io/hydration-override: false
```

If kpt sees `config.kubernetes.io/hydration-override: false`, it won't allow the functions to modify the resource. The user can add 
this annotation to every resource that they don't want functions to modify.

Pros:
- The user would not need to write an exclusion field for each function.

Cons:
- It will not be possible to control at a function level. If users want some functions to modify the Kptfile/meta resources, and others not to touch it, 
they will have to re-organize their package.


### Use the existing selector mechanism

We can recommend that users use the existing selectors to exclude the Kptfile and functionConfigs if they don't want functions to run on them. 
Here is an example: https://github.com/natasha41575/kpt-exclude-kptfile/blob/main/Kptfile. The Kptfile looks like:

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: example
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/set-labels:unstable
    configPath: functionconfig.yaml
    selectors:
    - apiVersion: "batch/v1"
    - apiVersion: "apps/v1"
    - apiVersion: "v1"
    - apiVersion: "example.com/v1"
```

Some problems with this approach:
- It is verbose.
- Verbosity increases if the user has more resources in the package.
- The user will have to keep track of an exhaustive list of all their resources in their package just to exclude the Kptfile.
- kpt and kpt functions should already know about the Kptfile, so this overhead on the user's side is unnecessary.

### Rely on the existing annotation config.kubernetes.io/local-config

We can exclude all local-config resources (i.e. resources with the annotation `config.kubernetes.io/local-config: true`) from functions. This annotation is 
currently used by kpt live apply to determine whether to apply the resource to the cluster. However, users may have some resources that they want kpt to modify or be 
available as function inputs, but that they don't want to be applied to the cluster. In these cases, overloading the annotation in this way will
break some functions.

### Mark meta resources with an annotation
Kpt can add the annotation config.kubernetes.io/meta-resource: true to all meta resources, to allow users to easily target the Kptfile and functionConfigs. 
Users can then use the annotation selector mechanism to target meta resources if desired.

However, because most meta resources already have the `config.kubernetes.io/local-config: true` annotation, a new meta resource annotation is not necessary. 
With the proposed `exclude` mechanism, users can filter out local-config easily, and in other cases, Kind- based exclusion is good enough.

