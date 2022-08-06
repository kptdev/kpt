---
title: "`eval`"
linkTitle: "eval"
type: docs
description: >
  Execute function on resources
---

<!--mdtogo:Short
    Execute function on resources
-->

`eval` executes a function on resources in a directory. Functions are packaged
as container images.

If the function fails (i.e. exits with non-zero status code), `eval` will abort
and the local filesystem is left intact.

Refer to the [Imperative Function Execution] for detailed overview.

### Synopsis

<!--mdtogo:Long-->

```
kpt fn eval [DIR|-] [flags] [-- fn-args]
```

#### Args

```
DIR|-:
  Path to the local directory containing resources. Defaults to the current
  working directory. Using '-' as the directory path will cause `eval` to
  read resources from `stdin` and write the output to `stdout`. When resources are
  read from `stdin`, they must be in one of the following input formats:

  1. Multi object YAML where resources are separated by `---`.

  2. KRM Function Specification wire format where resources are wrapped in an object
     of kind ResourceList.

  If the output is written to `stdout`, resources are written in multi object YAML
  format where resources are separated by `---`.
```

```
fn-args:
  function arguments to be provided as input to the function. These must be
  provided in the `key=value` format and come after the separator `--`.
```

#### Flags

```
--as-current-user:
  Use the `uid` and `gid` of the kpt process for container function execution.
  By default, container function is executed as `nobody` user. You may want to use
  this flag to run higher privilege operations such as mounting the local filesystem.

--env, e:
  List of local environment variables to be exported to the container function.
  By default, none of local environment variables are made available to the
  container running the function. The value can be in `key=value` format or only
  the key of an already exported environment variable.

--exec:
  Path to the local executable binary to execute as a function. Quotes are needed
  if the executable requires arguments. `eval` executes only one function, so do
  not use `--image` flag with this flag. This is useful for testing function locally
  during development. It enables faster dev iterations by avoiding the function to
  be published as container image.

--fn-config:
  Path to the file containing `functionConfig` for the function.

--image, i:
  Container image of the function to execute e.g. `gcr.io/kpt-fn/set-namespace:v0.1`.
  For convenience, if full image path is not specified, `gcr.io/kpt-fn/` is added as default prefix.
  e.g. instead of passing `gcr.io/kpt-fn/set-namespace:v0.1` you can pass `set-namespace:v0.1`.
  `eval` executes only one function, so do not use `--exec` flag with this flag.

--image-pull-policy:
  If the image should be pulled before rendering the package(s). It can be set
  to one of always, ifNotPresent, never. If unspecified, always will be the
  default.
  If using always, kpt will ensure the function images to run are up-to-date
  with the remote container registry. This can be useful for tags like v1.
  If using ifNotPresent, kpt will only pull the image when it can't find it in
  the local cache.
  If using never, kpt will only use images from the local cache.

--include-meta-resources, m:
  (DEPRECATED) include-meta-resources is no longer necessary because meta
  resources are included by default with kpt version v1.0.0-beta.15+.

--match-api-version:
  Select resources matching the given apiVersion.

--match-kind
  Select resources matching the given kind.

--match-name:
  Select resources matching the given name.
  
--match-namespace:
  Select resources matching the given namespace.

--mount:
  List of storage options to enable reading from the local filesytem. By default,
  container functions can not access the local filesystem. It accepts the same options
  as specified on the [Docker Volumes] for `docker run`. All volumes are mounted
  readonly by default. Specify `rw=true` to mount volumes in read-write mode.

--network:
  If enabled, container functions are allowed to access network.
  By default it is disabled.

--output, o:
  If specified, the output resources are written to provided location,
  if not specified, resources are modified in-place.
  Allowed values: stdout|unwrap|<OUT_DIR_PATH>
  1. stdout: output resources are wrapped in ResourceList and written to stdout.
  2. unwrap: output resources are written to stdout, in multi-object yaml format.
  3. OUT_DIR_PATH: output resources are written to provided directory.
     The provided directory must not already exist.

--type, t;
  Specify the function type. Accept value `mutator` (default), `validator`. 
  If used with `--save`, this flag will save the evaluated function to the corresponding
   Kptfile section: `.pipeline.mutators` if type is `mutator`; `.pipeline.validators` if type
    is `validator`.

--results-dir:
  Path to a directory to write structured results. Directory will be created if
  it doesn't exist. Structured results emitted by the functions are aggregated and saved
  to `results.yaml` file in the specified directory.
  If not specified, no result files are written to the local filesystem.
  
--save, s:
  Save the function image and fn-config to Kptfile. Require ` + "`" + `--image` + "`" + `.
  
```

#### Environment Variables

```
KPT_FN_RUNTIME:
  The runtime to run kpt functions. It must be one of "docker", "podman" and "nerdctl".
```

<!--mdtogo-->

## Examples

<!--mdtogo:Examples-->

```shell
# execute container my-fn on the resources in DIR directory and
# write output back to DIR
$ kpt fn eval DIR -i gcr.io/example.com/my-fn
```

```shell
# execute container my-fn on the resources in DIR directory with
# `functionConfig` my-fn-config
$ kpt fn eval DIR -i gcr.io/example.com/my-fn --fn-config my-fn-config
```

```shell
# execute container my-fn with an input ConfigMap containing `data: {foo: bar}`
$ kpt fn eval DIR -i gcr.io/example.com/my-fn:v1.0.0 -- foo=bar
```

```shell
# execute container my-fn and save it to Kptfile `pipeline.mutators` (Default) list.
$ kpt fn eval DIR -s -i gcr.io/example.com/my-fn:v1.0.0 -- foo=bar
```

```shell
# execute container my-fn and save it to Kptfile `pipeline.validators` list.
$ kpt fn eval DIR -s -t validator -i gcr.io/example.com/my-fn:v1.0.0 -- foo=bar
```

```shell
# execute executable my-fn on the resources in DIR directory and
# write output back to DIR
$ kpt fn eval DIR --exec ./my-fn
```

```shell
# execute executable my-fn with arguments on the resources in DIR directory and
# write output back to DIR
$ kpt fn eval DIR --exec "./my-fn arg1 arg2"
```

```shell
# execute container my-fn on the resources in DIR directory,
# save structured results in /tmp/my-results dir and write output back to DIR
$ kpt fn eval DIR -i gcr.io/example.com/my-fn --results-dir /tmp/my-results-dir
```

```shell
# execute container my-fn on the resources in DIR directory with network access enabled,
# and write output back to DIR
$ kpt fn eval DIR -i gcr.io/example.com/my-fn --network
```

```shell
# execute container my-fn on the resource in DIR and export KUBECONFIG
# and foo environment variable
$ kpt fn eval DIR -i gcr.io/example.com/my-fn --env KUBECONFIG -e foo=bar
```

```shell
# execute kubeval function by mounting schema from a local directory on wordpress package
$ kpt fn eval -i gcr.io/kpt-fn/kubeval:v0.1 \
  --mount type=bind,src="/path/to/schema-dir",dst=/schema-dir \
  --as-current-user wordpress -- additional_schema_locations=/schema-dir
```

```shell
# chaining functions using the unix pipe to set namespace and set labels on
# wordpress package
$ kpt fn source wordpress \
  | kpt fn eval - -i gcr.io/kpt-fn/set-namespace:v0.1 -- namespace=mywordpress \
  | kpt fn eval - -i gcr.io/kpt-fn/set-labels:v0.1 -- label_name=color label_value=orange \
  | kpt fn sink wordpress
```

```shell
# execute container 'set-namespace' on the resources in current directory and write
# the output resources to another directory
$ kpt fn eval -i gcr.io/kpt-fn/set-namespace:v0.1 -o path/to/dir -- namespace=mywordpress
```

```shell
# execute container 'set-namespace' on the resources in current directory and write
# the output resources to stdout which are piped to 'kubectl apply'
$ kpt fn eval -i gcr.io/kpt-fn/set-namespace:v0.1 -o unwrap -- namespace=mywordpress \
| kubectl apply -f -
```

```shell
# execute container 'set-namespace' on the resources in current directory and write
# the wrapped output resources to stdout which are passed to 'set-annotations' function
# and the output resources after setting namespace and annotation is written to another directory
$ kpt fn eval -i gcr.io/kpt-fn/set-namespace:v0.1 -o stdout -- namespace=staging \
| kpt fn eval - -i gcr.io/kpt-fn/set-annotations:v0.1.3 -o path/to/dir -- foo=bar
```

```shell
# execute container 'set-namespace' on the resources with 'name' foo and 'kind' Deployment
# in current directory
kpt fn eval -i set-namespace:v0.1 --by-kind Deployment --by-name foo -- namespace=staging
```

```shell
# execute container my-fn with podman on the resources in DIR directory and
# write output back to DIR
$ KPT_FN_RUNTIME=podman kpt fn eval DIR -i gcr.io/example.com/my-fn
```

<!--mdtogo-->

[docker volumes]: https://docs.docker.com/storage/volumes/
[imperative function execution]: /book/04-using-functions/02-imperative-function-execution
[function specification]: /book/05-developing-functions/01-functions-specification
