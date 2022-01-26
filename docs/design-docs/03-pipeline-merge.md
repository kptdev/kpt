# Pipeline Merge

* Author(s): Phani Teja Marupaka, phanimarupaka
* Approver: Sunil Arora, Mike Borozdin

## Why

Currently, `kpt pkg update` doesn't merge the `pipeline` section in the Kptfile as expected. 
The fact that the `pipeline` section is a non-associative list with no defined function identity makes 
it very difficult to merge with upstream counterparts. Ordering of the functions is also important.
This friction is forcing users to use `setters` and discouraging them from declaring other functions in the `pipeline` as 
they will be deleted during the `kpt pkg update`. [Here](https://github.com/GoogleContainerTools/kpt/issues/2529) 
is the example issue. Merging the pipeline correctly will reduce huge amounts of 
friction in declaring new functions. This will encourage users to declare more functions 
in the pipeline which in turn helps to **avoid excessive parameterization**.

Consider the example of [Landing Zone](https://github.com/GoogleCloudPlatform/blueprints/tree/main/catalog) blueprints. 
Parameters(setters) are the primary interface for the package. This is an anti-pattern for package-as-interface motivation 
and one of the major blockers for not using other functions is the merge behavior 
of the pipeline. If this problem is solved, LZ maintainers can rewrite the packages 
with best practices aligned to the bigger goal of treating the package-as-interface, 
discourage excessive use of setters and only use them as parameterization techniques of last resort.

## Design

In order to solve this problem, we should merge the pipeline section of the Kptfile 
in a custom manner based on the most common use-cases and the expected outputs in 
such scenarios. There are no user interface changes. Users will invoke the `kpt pkg 
update` command in the same way they do currently. This effort will improve the merged 
output of the pipeline section.

**Is this change backwards compatible**: We are not making any changes to the api, 
we are only improving the merged output of the `pipeline` section. This change will 
produce a different output of Kptfile when compared to the current version but this 
is not a breaking change.

## User guide

Here is what users can expect when they invoke the command, `kpt pkg update` on a package.

#### Terminology

**Original**: Source of the last fetched version of the package, represented by the upstreamLock section in Kptfile.

**Updated upstream**: Declared source of the updated package, represented by the upstream section in Kptfile.

**Local**: Local fork of the package on disk.

Firstly, we need to define the identity of the function in order to uniquely identify 
a function across three sources to perform a 3-way merge. We will introduce a 
new field `name` for identifying functions and merge pipeline as associative list.
The aim is to encourage users eventually to have `name` field specified for all functions
similar to containers in deployment. But in the meanwhile, we will be using image name in 
order to identify the function and make it an associative list for merging. The limitation
of this approach is the multiple functions with same image can be declared in the pipeline.
In that case we will fall back to current merge logic of merging pipeline as non-associative list.
In order to have deterministic behavior, we will use image as merge key iff none of the 
functions have `name` field specified and iff there are no duplicate image names declare in functions.

So the algorithm to merge:

1. If `name` field is specified in all the functions across all sources, merge 
it as associative list using `name` field as merge key.
2. If `name` field is not specified in all the functions across all sources,
and if there is no duplicate declaration of image names for functions, use `image` value
(excluding the version) as merge key and merge it as associative list.
3. In all other cases fall back to the current merge behavior.

Here is an example of the merging apply-setters function when new setters are added 
upstream and existing setter values are updated locally. This is the most common 
use case where kpt fails to merge them. Since `name` field is not specified, `image`
value is used to identify and merge the function

```
Original
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configMap:
        image: nginx
        tag: 1.0.1
```
```
Updated upstream
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configMap:
        image: nginx
        tag: 1.0.1
        new-setter: new-setter-value // new setter is added
```
```
Local
```

```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configMap:
        image: nginx
        tag: 1.2.0 // value of tag is updated
```
```
Current Output
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configMap:
        image: nginx
        tag: 1.0.1 // entire pipeline is overridden by upstream 
        new-setter: new-setter-value
```
```
Expected Output
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configMap:
        image: nginx
        tag: 1.2.0 // updated tag is preserved
        new-setter: new-setter-value // new setter is pulled
```

In the above scenario, the value of the setter tag is not updated upstream, so 
the modified local value will be preserved. But in the following example, both 
upstream and local values change. So, similar to merging resources, upstream 
value wins if the same fields in both upstream and local are updated.

```
Original
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configPath: labels.yaml
```
```
Updated upstream
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configPath: labels-updated.yaml // upstream value changed
```
```
Local
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configPath: labels-local.yaml // local value changed
```
```
Expected Output
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configPath: labels-updated.yaml // upstream overrides local 
```

Similarly, the upstream version wins if both upstream and local are updated, else local is preserved.

```
Original
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-annotations:v0.1
      configPath: annotations.yaml
```
```
Updated upstream
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-annotations:v0.2.0
      configPath: annotations.yaml
```
```
Local
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-annotations:v0.1.1
      configPath: annotations.yaml
```
```
Expected Output
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-annotations:v0.2.0
      configPath: annotations.yaml
```

#### More examples with expected output

Newly added upstream function.

```
Original
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
```
```
Updated upstream
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
    - image: gcr.io/kpt-fn/generate-folders:v0.1
```
```
Local
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
    - image: gcr.io/kpt-fn/set-namespace:v0.1
      configMap:
        namespace: foo
```
```
Expected output
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
    - image: gcr.io/kpt-fn/set-namespace:v0.1
      configMap:
        namespace: foo
    - image: gcr.io/kpt-fn/generate-folders:v0.1
```

If a function is deleted upstream and not changed on the local, it will be deleted on local.

```
Original
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
    - image: gcr.io/kpt-fn/generate-folders:v0.1
```
```
Updated upstream
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
```
```
Local
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
    - image: gcr.io/kpt-fn/generate-folders:v0.1
    - image: gcr.io/kpt-fn/set-namespace:v0.1
      configMap:
        namespace: foo
```
```
Expected output
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
    - image: gcr.io/kpt-fn/set-namespace:v0.1
      configMap:
        namespace: foo
```

Same function declared multiple times: If the same function is declared multiple 
times and `name` field is not specified, fall back to the current merge behavior

```
Original
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
```
```
Updated upstream
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar-new
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${updated-setter-name}
```
```
Local
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/generate-folders:v0.1
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
        app: db
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: YOUR_TEAM
        put-value: my-team
```
```
Expected output
```
```yaml
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/search-replace:v0.1
    configMap:
      by-value: foo
      put-value: bar-new
  - image: gcr.io/kpt-fn/search-replace:v0.1
    configMap:
      by-value: abc
      put-comment: ${updated-setter-name}
```

In the following scenario, only one function has the `name` field specified. All the
other functions doesn't have name specified, hence we fall back to current merge logic.

```
Original
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
```
```
Updated upstream
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar-new
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${updated-setter-name}
```
```
Local
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: my-new-function
      configMap:
        by-value: YOUR_TEAM
        put-value: my-team
    - image: gcr.io/kpt-fn/generate-folders:v0.1
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
        app: db
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
```
```
Expected output
```
```yaml
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/search-replace:v0.1
    configMap:
      by-value: foo
      put-value: bar-new
  - image: gcr.io/kpt-fn/search-replace:v0.1
    configMap:
      by-value: abc
      put-comment: ${updated-setter-name}
```

Here is the ideal scenario which leads to deterministic merge behavior using `name`
field across all sources

```
Original
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr2
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
```
```
Updated upstream
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr1
      configMap:
        by-value: foo
        put-value: bar-new
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr2
      configMap:
        by-value: abc
        put-comment: ${updated-setter-name}
```
```
Local
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: my-new-function
      configMap:
        by-value: YOUR_TEAM
        put-value: my-team
    - image: gcr.io/kpt-fn/generate-folders:v0.1
      name: gf1
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/set-labels:v0.1
      name: sl1
      configMap:
        app: db
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr2
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
```
```
Expected output
```
```yaml
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/search-replace:v0.1
    name: my-new-function
    configMap:
      by-value: YOUR_TEAM
      put-value: my-team
  - image: gcr.io/kpt-fn/generate-folders:v0.1
    name: gf1
  - image: gcr.io/kpt-fn/search-replace:v0.1
    name: sr1
    configMap:
      by-value: foo
      put-value: bar-new
  - image: gcr.io/kpt-fn/set-labels:v0.1
    name: sl1
    configMap:
      app: db
  - image: gcr.io/kpt-fn/search-replace:v0.1
    name: sr2
    configMap:
      by-value: abc
      put-comment: ${updated-setter-name}
```

Merging selectors is difficult as there is no identity. If both upstream and
local selectors for a given function diverge, the entire section of selectors
from upstream will override the selectors on local for that function.

```
Origin
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/ensure-name-substring:v0.1
      selectors:
        - kind: Deployment
          name: wordpress
        - kind: Service
          name: wordpress
```
```
Updated upstream
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/ensure-name-substring:v0.1
      selectors:
        - kind: Deployment
          name: wordpress
        - kind: Service
          name: wordpress
        - kind: Foo
          name: wordpress
```
```
Local
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/ensure-name-substring:v0.1
      selectors:
        - kind: Deployment
          name: my-wordpress
        - kind: Service
          name: my-wordpress
        - namespace: my-space
```
```
Expected output
```
```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
    - image: gcr.io/kpt-fn/set-namespace:v0.1
      configMap:
        namespace: foo
    - image: gcr.io/kpt-fn/generate-folders:v0.1
```

## Open issues

 https://github.com/GoogleContainerTools/kpt/issues/2529

## Alternatives Considered

For identifying the function, we can add the function version to the primary key
(in addition to the function name). But it is highly likely that 
changing the function version means updating the function as opposed to adding a new function.

Why should upstream win in case of conflicts ? Is this what the user always expects?
- Not necessarily. User expectations might be different in different scenarios for resolving conflicts. 
But since we already went down the path of upstream-wins strategy in case of conflicts for merging resources, 
we are going down that route to maintain consistency.
- There is an open [issue](https://github.com/GoogleContainerTools/kpt/issues/1437) to support 
multiple conflict resolution strategies and provide interactive update behavior to resolve 
conflicts which is out of scope for this doc and will be addressed soon.
