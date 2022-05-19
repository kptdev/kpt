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

?> Refer to the [eval command reference][eval-doc] for usage.

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

## Specifying `functionConfig`

There are two ways to specify the `functionConfig`.

### `fn-config` flag

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

### CLI arguments

Many functions take a `functionConfig` of kind `ConfigMap` since they only need
simple key/value pairs as argument. For convenience, there is a way to provide
the key/value pairs as command line arguments. The following is equivalent to
what we showed previously:

```shell
$ kpt fn eval wordpress -i set-namespace:v0.1 -- namespace=mywordpress
```

Note that the arguments must come after the separator `--`.

## Specifying `selectors`

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

## Specifying `exclusions`

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

## Privileged Execution

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

### Network Access

By default, functions cannot access the network. You can enable network access
using the `--network` flag.

For example, `kubeval` function can download a JSON schema file:

```shell
$ kpt fn eval wordpress -i kubeval:v0.1 --network -- schema_location="https://kubernetesjsonschema.dev"
```

### Mounting Directories

By default, functions cannot access the host file system. You can use the
`--mount` flag to mount host volumes. kpt accepts the same options to `--mount`
specified on the [Docker Volumes] page.

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

## Chaining functions using the Unix pipe

As an alternative to declaring a pipeline in the `Kptfile`, you can chain
functions using the Unix pipe.

Here is an example:

```shell
$ kpt fn source wordpress \
  | kpt fn eval - -i set-namespace:v0.1 -- namespace=mywordpress \
  | kpt fn eval - -i set-labels:v0.1 -- app=wordpress env=prod \
  | kpt fn sink my-wordpress
```

?> Refer to the command reference for usage of [source][source-doc] and
[sink][sink-doc] commands.

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

[eval-doc]: /reference/cli/fn/eval/
[source-doc]: /reference/cli/fn/source/
[sink-doc]: /reference/cli/fn/sink/
[docker volumes]: https://docs.docker.com/storage/volumes/
