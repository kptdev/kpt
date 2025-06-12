---
title: "Chapter 4: Using functions"
linkTitle: "Chapter 4: Using functions"
description: |
    [Chapter 2](../02-concepts/#functions) provided a high-level conceptual explanation of functions. We also saw examples of how to use
    `fn eval` and `fn render` to execute functions. In this chapter, we will take a closer look at how to execute
    functions using these two approaches.

toc: true
menu:
  main:
    parent: "Book"
    weight: 40
---

## Declarative function execution

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

 > Refer to the [render command reference](../../reference/cli/fn/render/) for usage.

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

### Specifying `function`

#### `image`

The `image` field specifies the container image for the function. You can specify
an image from any container registry. If the registry is omitted, the default
container registry for functions catalog (`gcr.io/kpt-fn`) is prepended automatically.
For example, `set-labels:v0.1` is automatically expanded to `gcr.io/kpt-fn/set-labels:v0.1`.

#### `exec`

The `exec` field specifies the executable command for the function. You can specify
an executable with arguments.

Example below uses `sed` executable to replace all occurrences of `foo` with `bar`
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

### Specifying `functionConfig`

In [Chapter 2](../02-concepts/#functions), we saw this conceptual representation of a function invocation:

![img](/static/images/func.svg)

`functionConfig` is an optional meta resource containing the arguments to a
particular invocation of the function. There are two different ways to declare
the `functionConfig`.

#### `configPath`

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

#### `configMap`

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

### Specifying function `name`

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

### Specifying `selectors`

In some cases, you want to invoke the function only on a subset of resources based on a
selection criteria. This can be accomplished using selectors. At a high level, selectors
work as follows:

![img](/images/func-target.svg)

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

#### Specifying exclusions

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

## Imperative function execution

The `fn eval` command enables you to execute a single function without declaring
it in the package. This is referred to as imperative function execution.

For example, to set the namespace of all resources in the wordpress package
hierarchy:

```shell
$ kpt fn eval wordpress --image gcr.io/kpt-fn/set-namespace:v0.1 -- namespace=mywordpress
```

Alternatively, for convenience, you can use the short-hand form of the above command:

```shell
$ kpt fn eval wordpress -i set-namespace:v0.1 -- namespace=mywordpress
```

 > Refer to the [eval command reference](../../reference/cli/fn/eval/) for usage.

This changes the resources in the `wordpress` package and the `mysql`
subpackage.

For comparison, this has the same effect as the following declaration:

```yaml
# wordpress/Kptfile (Excerpt)
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: wordpress
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-namespace:v0.1
      configMap:
        namespace: mywordpress
```

So when should you execute a function using `eval` instead of `render`?

When you have one of these use cases:

- Perform a one-time operation
- Execute a function from a CI/CD system on packages authored by other teams
- Develop shell scripts and chain functions with the Unix pipe (`|`)
- Execute the function with privilege (Not allowed by `render`)

We will cover these topics in detail.

### Specifying `functionConfig`

There are two ways to specify the `functionConfig`.

#### `fn-config` flag

The general way to provide a `functionConfig` of arbitrary kind (core or custom
resources), is to declare the resource in a separate file and use the
`fn-config` flag.

```shell
$ cat << EOF > /tmp/fn-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ns
data:
  namespace: mywordpress
EOF
```

```shell
$ kpt fn eval wordpress -i set-namespace:v0.1 --fn-config /tmp/fn-config.yaml
```

#### CLI arguments

Many functions take a `functionConfig` of kind `ConfigMap` since they only need
simple key/value pairs as argument. For convenience, there is a way to provide
the key/value pairs as command line arguments. The following is equivalent to
what we showed previously:

```shell
$ kpt fn eval wordpress -i set-namespace:v0.1 -- namespace=mywordpress
```

Note that the arguments must come after the separator `--`.

### Specifying `selectors`

Selectors can be used to target specific resources for a function execution.

For example, you can selectively add an annotation to the resources if it has kind
`Deployment` AND name `wordpress`:

```shell
$ kpt fn eval wordpress -i set-annotations:v0.1 --match-kind Deployment --match-name wordpress -- foo=bar
```

Here is the list of available selector matcher flags:

1. `match-api-version`
2. `match-kind`
3. `match-name`
4. `match-namespace`
5. `match-annotations`
6. `match-labels`

### Specifying `exclusions`

Exclusions can be used to exclude specific resources for a function execution.

For example, you can set the namespace of all resources in the wordpress package, 
except for the ones with the label `foo: bar`:

```shell
$ kpt fn eval wordpress -i set-namespace:v0.1 --exclude-labels foo=bar -- namespace=my-namespace
```

If you use multiple exclusions, it will exclude resources that match all provided exclusions. For
example, you can set the namespace of all resources, except for those that have both kind "Deployment" 
and name "nginx":

`$ kpt fn eval wordpress -i set-namespace:v0.1 --exclude-kind Deployment --exclude-name nginx -- namespace=my-namespace`

Here is the list of available exclusion flags:

1. `exclude-api-version`
2. `exclude-kind`
3. `exclude-name`
4. `exclude-namespace`
5. `exclude-annotations`
6. `exclude-labels`

### Privileged Execution

Since the function is provided explicitly by the user, `eval` can be more
privileged and low-level than a declarative invocation using `render`. For
example, it can have access to the host system.

In general, we recommend against having functions that require privileged access
to the host since they can only be executed imperatively and pose a challenge in
terms of security, correctness, portability and speed. If at all possible,
functions should be executed hermetically with all required dependencies either
passed in as KRM resources (input items or functionConfig) or included in the
container image. However, there are some legitimate use cases where the only
available option requires either network access or mounting a volume from the
host. In those situations, you can use `eval` as described below.

#### Network Access

By default, functions cannot access the network. You can enable network access
using the `--network` flag.

For example, `kubeval` function can download a JSON schema file:

```shell
$ kpt fn eval wordpress -i kubeval:v0.1 --network -- schema_location="https://kubernetesjsonschema.dev"
```

#### Mounting Directories

By default, functions cannot access the host file system. You can use the
`--mount` flag to mount host volumes. kpt accepts the same options to `--mount`
specified on the [Docker Volumes](https://docs.docker.com/storage/volumes/) page.

For example, `kubeval` function can consume a JSON schema file:

```shell
$ kpt fn eval -i kubeval:v0.1 --mount type=bind,src="/path/to/schema-dir",dst=/schema-dir --as-current-user wordpress -- schema_location=file:///schema-dir
```

Note that the `--as-current-user` flag may be required to run the function as
your uid instead of the default `nobody` to access the host filesystem.

All volumes are mounted readonly by default. Specify `rw=true` to mount volumes
in read-write mode.

```
--mount type=bind,src="/path/to/schema-dir",dst=/schema-dir,rw=true
```

### Chaining functions using the Unix pipe

As an alternative to declaring a pipeline in the `Kptfile`, you can chain
functions using the Unix pipe.

Here is an example:

```shell
$ kpt fn source wordpress \
  | kpt fn eval - -i set-namespace:v0.1 -- namespace=mywordpress \
  | kpt fn eval - -i set-labels:v0.1 -- app=wordpress env=prod \
  | kpt fn sink my-wordpress
```

 > Refer to the command reference for usage of [source](../..//reference/cli/fn/source/) and
[sink](../../reference/cli/fn/sink/) commands.

The following describes the above pipeline:

1. The `source` command is used to read the resources in the package hierarchy
   (`wordpress` and `mysql` packages). The output of the `source` command
   follows the KRM Function Specification standard, which we are going to look
   at in chapter 5.
2. The output of the `source` function is piped into the `set-namespace`
   function. `eval` function is instructed to read inputs items from the `stdin`
   using `-`. This is the convention used in all commands in kpt that can read
   from `stdin`. The `set-namespace` function mutates the input items and emits
   the output items.
3. The output of the `set-namespace` function is piped into `set-labels`
   function which adds the given labels to all resources.
4. The `sink` command writes the output of `set-labels` to the filesystem.

This is a low-level and less abstracted approach to executing functions. You can
instead write the output of the pipeline to a different directory instead of
mutating the directory in-place. You can also pipe to other programs (e.g.
`sed`, `yq`) that are not functions. Be mindful that the cost of this low-level
flexibility is not having benefits provided by functions: scalability,
reusability, and encapsulation.

## Function results

In kpt, the counterpart to Unix philsophophy of "everything is a file" is "everything is a
Kubernetes resource". This also extends to the results of executing functions using `eval` or
`render`. In addition to providing a human-readable terminal output, these commands provide
structured results which can be consumed by other tools. This enables you to build robust UI layers
on top of kpt. For example:

- Create a custom dashboard that shows the results returned by functions
- Annotate a GitHub Pull Request with results returned by a validator function at the granularity of individuals fields

In both `render` and `eval`, structured results can be enabled using the `--results-dir` flag.

For example:

```shell
$ kpt fn render wordpress --results-dir /tmp
Package "wordpress/mysql":

[PASS] "gcr.io/kpt-fn/set-labels:v0.1"

Package "wordpress":

[PASS] "gcr.io/kpt-fn/set-labels:v0.1"
[PASS] "gcr.io/kpt-fn/kubeval:v0.1"

Successfully executed 3 function(s) in 2 package(s).
For complete results, see /tmp/results.yaml
```

The results are provided as resource of kind `FunctionResultList`:

```yaml
# /tmp/results.yaml
apiVersion: kpt.dev/v1
kind: FunctionResultList
metadata:
  name: fnresults
exitCode: 0
items:
  - image: gcr.io/kpt-fn/set-labels:v0.1
    exitCode: 0
  - image: gcr.io/kpt-fn/set-labels:v0.1
    exitCode: 0
  - image: gcr.io/kpt-fn/kubeval:v0.1
    exitCode: 0
```

Let's see a more interesting result where the `kubeval` function catches a validation issue.
For example, change the value of `port` field in `service.yaml` from `80` to `"80"` and
rerun:

```shell
$ kpt fn render wordpress --results-dir /tmp
Package "wordpress/mysql":

[PASS] "gcr.io/kpt-fn/set-labels:v0.1"

Package "wordpress":

[PASS] "gcr.io/kpt-fn/set-labels:v0.1"
[FAIL] "gcr.io/kpt-fn/kubeval:v0.1"
  Results:
    [ERROR] Invalid type. Expected: integer, given: string in object "v1/Service/wordpress" in file "service.yaml" in field "spec.ports.0.port"
  Exit code: 1

For complete results, see /tmp/results.yaml
```

The results resource will now contain failure details:

```yaml
# /tmp/results.yaml
apiVersion: kpt.dev/v1
kind: FunctionResultList
metadata:
  name: fnresults
exitCode: 1
items:
  - image: gcr.io/kpt-fn/set-labels:v0.1
    exitCode: 0
  - image: gcr.io/kpt-fn/set-labels:v0.1
    exitCode: 0
  - image: gcr.io/kpt-fn/kubeval:v0.1
    exitCode: 1
    results:
      - message: "Invalid type. Expected: integer, given: string"
        severity: error
        resourceRef:
          apiVersion: v1
          kind: Service
          name: wordpress
        field:
          path: spec.ports.0.port
        file:
          path: service.yaml
```
