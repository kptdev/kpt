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

## Specifying `function`

### `image`

The `image` field specifies the container image for the function. You can specify
an image from any container registry. If the registry is omitted, the default
container registry for functions catalog (`gcr.io/kpt-fn`) is prepended automatically.
For example, `set-labels:v0.1` is automatically expanded to `gcr.io/kpt-fn/set-labels:v0.1`.

### `exec`

The `exec` field specifies the executable command for the function. You can specify
an executable with arguments.

Example below uses `sed` executable to replace all occurances of `foo` with `bar`
in the package resources.

```yaml
# PKG_DIR/Kptfile (Excerpt)
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: app
pipeline:
  mutators:
    - exec: "sed -e 's/foo/bar/'"
```

Note that you must render the package by allowing executables by specifying `--allow-exec`
command line flag as shown below.

```shell
$ kpt fn render [PKG_DIR] --allow-exec
```

Using `exec` is not recommended for two reasons:

- It makes the package non-portable since rendering the package requires the
  executables to be present on the system.
- Executing binaries is not very secure since they can perform privileged operations
  on the system.

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

## Specifying function `name`

Functions can optionally be named using the `pipeline.mutators.name`
or `pipeline.validators.name` field to identify a function.

For example:

```yaml
# wordpress/mysql/Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: mysql
pipeline:
  mutators:
    - name: set tier label
      image: set-labels:v0.1
      configMap:
        tier: mysql
```

Unique function names for all functions in the Kptfile function
pipeline is recommended.  If `name` is specified, `kpt pkg update`
will merge each function pipeline list as an associative list, using
`name` as the merge key. An unspecified `name` or duplicated names may
result in unexpected merges.

## Specifying `selectors`

In some cases, you want to invoke the function only on a subset of resources based on a
selection criteria. This can be accomplished using selectors. At a high level, selectors
work as follows:

![img](/static/images/func-target.svg)

Resources that are selected are passed as input to the function.
Resources that are not selected are passed through unchanged.

For example, let's add a function to the pipeline that adds an annotation to 
resources with name `wordpress-mysql` only:

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
        - name: wordpress-mysql
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
         app: wordpress
  validators:
    - image: gcr.io/kpt-fn/kubeval:v0.1
```

When you invoke the render command, the `mysql` package is rendered first, and `set-annotations`
function is invoked only on the resources with name `wordpress-mysql`. Then, `set-label`
function is invoked on all the resources in the package hierarchy of `wordpress` package.

```shell
$ kpt fn render wordpress
Package "wordpress/mysql": 
[RUNNING] "gcr.io/kpt-fn/set-label:v0.1"
[PASS] "gcr.io/kpt-fn/set-label:v0.1"

Package "wordpress": 
[RUNNING] "gcr.io/kpt-fn/set-annotations:v0.1" on 3 resource(s)
[PASS] "gcr.io/kpt-fn/set-annotations:v0.1"
[RUNNING] "gcr.io/kpt-fn/set-label:v0.1"
[PASS] "gcr.io/kpt-fn/set-label:v0.1"
[RUNNING] "gcr.io/kpt-fn/kubeval:v0.1"
[PASS] "gcr.io/kpt-fn/kubeval:v0.1"

Successfully executed 4 function(s) in 2 package(s).
```

As another example, let's add another function to the pipeline that adds a prefix to the name of a resource if:
- it has kind `Deployment` AND name `wordpress`
  **OR**
- it has kind `Service` AND name `wordpress`

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
        - name: wordpress-mysql
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
  validators:
    - image: gcr.io/kpt-fn/kubeval:v0.1
```

Now, let's render the package:

```shell
kpt fn render wordpress
Package "wordpress/mysql": 
[RUNNING] "gcr.io/kpt-fn/set-label:v0.1"
[PASS] "gcr.io/kpt-fn/set-label:v0.1"

Package "wordpress": 
[RUNNING] "gcr.io/kpt-fn/set-annotations:v0.1" on 3 resource(s)
[PASS] "gcr.io/kpt-fn/set-annotations:v0.1"
[RUNNING] "gcr.io/kpt-fn/set-label:v0.1"
[PASS] "gcr.io/kpt-fn/set-label:v0.1"
[RUNNING] "gcr.io/kpt-fn/ensure-name-substring:v0.1" on 2 resource(s)
[PASS] "gcr.io/kpt-fn/ensure-name-substring:v0.1"
[RUNNING] "gcr.io/kpt-fn/kubeval:v0.1"
[PASS] "gcr.io/kpt-fn/kubeval:v0.1"

Successfully executed 5 function(s) in 2 package(s).
```

Note that the `ensure-name-substring` function is applied only to the 
resources matching the selection criteria.

If you have resources with particular labels or annotations that you want to use to
select your resources, you can do so. For example, here is a function that will only
be applied to resources matching the label `foo: bar`:

```yaml
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
        - labels:
            foo: bar
  validators:
    - image: gcr.io/kpt-fn/kubeval:v0.1
```

The following are the matchers you can specify in a selector:

1. `apiVersion`: `apiVersion` field value of resources to be selected.
2. `kind`: `kind` field value of resources to be selected.
3. `name`: `metadata.name` field value of resources to be selected.
4. `namespace`: `metadata.namespace` field of resources to be selected.
5. `annotations`: resources with matching annotations will be selected.
6. `labels`: resources with matching labels will be selected.

### Specifying exclusions

Similar to `selectors`, you can also specify resources that should be excluded from functions.

For example, you can exclude a resource if it has both kind "Deployment" and name "nginx":

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: wordpress
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-annotations:v0.1
      configMap:
        tier: mysql
      exclude:
        - kind: Deployment
          name: nginx
  validators:
    - image: gcr.io/kpt-fn/kubeval:v0.1
```

This is distinct from the following, which excludes a resource if it has either kind "Deployment" or name "nginx":

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: wordpress
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-annotations:v0.1
      configMap:
        tier: mysql
      exclude:
        - kind: Deployment
        - name: nginx
  validators:
    - image: gcr.io/kpt-fn/kubeval:v0.1
```

The following are the matchers you can specify in an exclusion:

1. `apiVersion`: `apiVersion` field value of resources to be excluded.
2. `kind`: `kind` field value of resources to be excluded.
3. `name`: `metadata.name` field value of resources to be excluded.
4. `namespace`: `metadata.namespace` field of resources to be excluded.
5. `annotations`: resources with matching annotations will be excluded.
6. `labels`: resources with matching labels will be excluded.

[chapter 2]: /book/02-concepts/03-functions
[render-doc]: /reference/cli/fn/render/
[Package identifier]: book/03-packages/01-getting-a-package?id=package-name-and-identifier
