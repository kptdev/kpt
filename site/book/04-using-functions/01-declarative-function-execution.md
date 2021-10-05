In many real-world scenarios, it's not sufficient to only have packages of
static, fully-rendered resource configuration. You want the package to declare
both static data as well as operations that should be performed on current
resources and any resource that may be added in the future as you edit the
package. Example use cases:

- Set the namespace on all namespace-scoped resources
- Always perform schema validation on resources in the package
- Always enforce a constraint policy on resources in the package
- Generate resources using a human-authored custom resource

In kpt, this is achieved by declaring a pipeline of functions in the `Kptfile`
and executing all the pipelines in the package hierarchy in a depth-first order
using the `fn render` command.

In our wordpress example, the top-level `wordpress` package declares this
pipeline:

```yaml
# wordpress/Kptfile (Excerpt)
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: wordpress
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
        app: wordpress
  validators:
    - image: gcr.io/kpt-fn/kubeval:v0.1
```

This declares two functions:

- `set-label` is a mutator function which adds a set of labels to resources.
- `kubeval` is a validator function which validates the resources against their
  OpenAPI schema.

?> Refer to the [Functions Catalog](https://catalog.kpt.dev/ ":target=_self")
for details on how to use a particular function.

There are two differences between mutators and validators:

1. Validators are not allowed to modify resources.
2. Validators are always executed after mutators.

The `mysql` subpackage declares only a mutator function:

```yaml
# wordpress/mysql/Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: mysql
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
        tier: mysql
```

Now, let's render the package hierarchy:

```shell
$ kpt fn render wordpress
Package "wordpress/mysql":

[PASS] "gcr.io/kpt-fn/set-labels:v0.1"

Package "wordpress":

[PASS] "gcr.io/kpt-fn/set-labels:v0.1"
[PASS] "gcr.io/kpt-fn/kubeval:v0.1"

Successfully executed 3 function(s) in 2 package(s).
```

?> Refer to the [render command reference][render-doc] for usage.

When you invoke the `render` command, kpt performs the following steps:

1. Sequentially executes the list of mutators declared in the `mysql` package.
   The input to the first function is the set of resources read from the
   configuration files in the `mysql` package. The output of the first function
   is the input of the second function and so on.
2. Similarly, executes all the validators declared in the `mysql` package. The
   input to the first validator is the output of the last mutator. The output of
   the last validator is the output of the pipeline in the `mysql` package.
3. Sequentially executes the list of mutators declared in the `wordpress`
   package. The input to the first function is the union of:

   - Resources read from configuration files in the `wordpress` package AND
   - Output of the pipeline from the `mysql` package (Step 2).

4. Similarly, execute all the validators declared in the `wordpress` package.
   The output of the last validator is the output of the pipeline in the
   `wordpress` package.
5. Write the output of step 4 by modifying the local filesystem in-place. This
   can change both `wordpress` and `mysql` packages.

The end result is that:

1. Resources in the `mysql` package are labelled with `tier: mysql`.
2. Resources in `mysql` and `wordpress` packages are labelled with
   `app: wordpress`.
3. Resources in `mysql` and `wordpress` packages are validated against their
   OpenAPI spec.

If any of the functions in the pipeline fails for whatever reason, then the
entire pipeline is aborted and the local filesystem is left intact.

## Specifying `image`

The `image` field specifies the container image for the function. You can specify
an image from any container registry. If the registry is omitted, the default
container registry for functions catalog (`gcr.io/kpt-fn`) is prepended automatically.
For example, `set-labels:v0.1` is automatically expanded to `gcr.io/kpt-fn/set-labels:v0.1`.

## Specifying `functionConfig`

In [Chapter 2], we saw this conceptual representation of a function invocation:

![img](/static/images/func.svg)

`functionConfig` is an optional meta resource containing the arguments to a
particular invocation of the function. There are two different ways to declare
the `functionConfig`.

### `configPath`

The general way to provide a `functionConfig` of arbitrary kind (core or custom
resources), is to declare the resource in a separate file in the same directory
as the `Kptfile` and refer to it using the `configPath` field.

For example:

```yaml
# wordpress/mysql/Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: mysql
pipeline:
  mutators:
    - image: set-labels:v0.1
      configPath: labels.yaml
```

```yaml
# wordpress/mysql/labels.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: labels
data:
  tier: mysql
```

### `configMap`

Many functions take a `functionConfig` of kind `ConfigMap` since they only need
simple key/value pairs as argument. For convenience, there is a way to inline
the key/value pairs in the `Kptfile`.

The following is equivalent to what we showed before:

```yaml
# wordpress/mysql/Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: mysql
pipeline:
  mutators:
    - image: set-labels:v0.1
      configMap:
        tier: mysql
```

## Specifying `selectors`

Selectors can be used to target specific resources for a function execution. 
Some of the example use-cases are:
1. Run a function on all deployments in `mysql` subpackage.
2. Run a function on all deployments and services in the `wordpress` package.
3. Run a function on all GCS bucket resources with namespace `my-ns`.

Selectors follow OR of AND(s) approach where, within each selector, the selection 
properties are ANDed and the selected resources are UNIONed with other selected resources.

Example 1: Add annotations only to the `mysql` subpackage resources but add labels to all resources
in `wordpress` package:

```yaml
# wordpress/Kptfile (Excerpt)
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: wordpress
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-annotations:v0.1
      configMap:
        tier: mysql
      selectors:
        - packagePath: mysql
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
         app: wordpress
```

When you invoke the render command, the `mysql` package is hydrated first, and `set-annotations`
function is invoked only on the resources from `mysql` package. Then, `set-label`
function is invoked on all the resources in the directory tree of `wordpress` package.

```shell
$ kpt fn render wordpress
Package "wordpress/mysql": 
[RUNNING] "gcr.io/kpt-fn/set-label:v0.1"
[PASS] "gcr.io/kpt-fn/set-label:v0.1" in 4.8s

Package "wordpress": 
[RUNNING] "gcr.io/kpt-fn/set-annotations:v0.1" on 3 resource(s)
[PASS] "gcr.io/kpt-fn/set-annotations:v0.1" in 3.1s
[RUNNING] "gcr.io/kpt-fn/set-label:v0.1"
[PASS] "gcr.io/kpt-fn/set-label:v0.1" in 3s
[RUNNING] "gcr.io/kpt-fn/kubeval:v0.1"
[PASS] "gcr.io/kpt-fn/kubeval:v0.1" in 3.2s

Successfully executed 4 function(s) in 2 package(s).
```

Example 2: Add another function to pipeline to set name-prefix to only `Deployment` 
resources with specific `name` OR `Service` resources with specific `name`

```yaml
# wordpress/Kptfile (Excerpt)
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: wordpress
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-annotations:v0.1
      configMap:
        tier: mysql
      selectors:
        - packagePath: mysql
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
        app: wordpress
    - image: gcr.io/kpt-fn/ensure-name-substring:v0.1
      configMap:
        prepend: dev-
      selectors:
        - kind: Deployment
          name: wordpress
        - kind: Service
          name: wordpress
```

Now, let's render the package hierarchy:

```shell
kpt fn render wordpress
Package "wordpress/mysql": 
[RUNNING] "gcr.io/kpt-fn/set-label:v0.1"
[PASS] "gcr.io/kpt-fn/set-label:v0.1" in 4.3s

Package "wordpress": 
[RUNNING] "gcr.io/kpt-fn/set-annotations:v0.1" on 3 resource(s)
[PASS] "gcr.io/kpt-fn/set-annotations:v0.1" in 3s
[RUNNING] "gcr.io/kpt-fn/set-label:v0.1"
[PASS] "gcr.io/kpt-fn/set-label:v0.1" in 2.9s
[RUNNING] "gcr.io/kpt-fn/ensure-name-substring:v0.1" on 2 resource(s)
[PASS] "gcr.io/kpt-fn/ensure-name-substring:v0.1" in 2.9s
[RUNNING] "gcr.io/kpt-fn/kubeval:v0.1"
[PASS] "gcr.io/kpt-fn/kubeval:v0.1" in 3.4s

Successfully executed 5 function(s) in 2 package(s).
```
Note that the `ensure-name-substring` function is applied only to the 
resources matching the input selection criteria .

Here are the list of available selector properties:

1. `apiVersion`: `apiVersion` field value of resources to be selected.
2. `kind`: `kind` field value of resources to be selected.
3. `name`: `metadata.name` field value of resources to be selected.
4. `namespace`: `metadata.namespace` field of resources to be selected.
5. `packagePath`: PackagePath of resources to be selected. The path must be
   OS-agnostic Slash-separated relative to the package directory. Examples
   - `packagePath: mysql` - selects resources in the `mysql` subpackage excluding resources of nested subpackages of `mysql`.
   - `packagePath: .` - selects resources in current package excluding resources in subpackages of current package.

[chapter 2]: /book/02-concepts/03-functions
[render-doc]: /reference/cli/fn/render/
