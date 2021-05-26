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

If the function fails (i.e. exits with non-zero status code), `eval` will
abort and the local filesystem is left intact.

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

  2. `Function Specification` wire format where resources are wrapped in an object
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

--dry-run:
  If enabled, the resources are not written to local filesystem, instead they
  are written to stdout. By defaults it is disabled.
  
--env, e:
  List of local environment variables to be exported to the container function.
  By default, none of local environment variables are made available to the
  container running the function. The value can be in `key=value` format or only
  the key of an already exported environment variable.

--exec-path:
  Path to the local executable binary to execute as a function. `eval` executes
  only one function, so do not use `--image` flag with this flag. This is useful
  for testing function locally during development. It enables faster dev iterations
  by avoiding the function to be published as container image.
  
--fn-config:
  Path to the file containing `functionConfig` for the function.

--image:
  Container image of the function to execute e.g. `gcr.io/kpt-fn/set-namespace:v0.1`.
  `eval` executes only one function, so do not use `--exec-path` flag with this flag.

--image-pull-policy:
  If the image should be pulled before rendering the package(s). It can be set
  to one of always, ifNotPresent, never. If unspecified, always will be the
  default.
  If using always, kpt will ensure the function images to run are up-to-date
  with the remote container registry. This can be useful for tags like v1.
  If using ifNotPresent, kpt will only pull the image when it can't find it in
  the local cache.
  If using never, kpt will only use images from the local cache.

--include-meta-resources:
  If enabled, meta resources (i.e. `Kptfile` and `functionConfig`) are included
  in the input to the function. By default it is disabled.

--mount:
  List of storage options to enable reading from the local filesytem. By default,
  container functions can not access the local filesystem. It accepts the same options
  as specified on the [Docker Volumes] for `docker run`. All volumes are mounted
  readonly by default. Specify `rw=true` to mount volumes in read-write mode.

--network:
  If enabled, container functions are allowed to access network.
  By default it is disabled.

--results-dir:
  Path to a directory to write structured results. Directory must exist.
  Structured results emitted by the functions are aggregated and saved
  to `results.yaml` file in the specified directory.
  If not specified, no result files are written to the local filesystem.
```

<!--mdtogo-->

## Examples
<!--mdtogo:Examples-->

```
# execute container my-fn on the resources in DIR directory and
# write output back to DIR
$ kpt fn eval DIR --image gcr.io/example.com/my-fn
```

```
# execute container my-fn on the resources in DIR directory with
# `functionConfig` my-fn-config
$ kpt fn eval DIR --image gcr.io/example.com/my-fn --fn-config my-fn-config
```

```
# execute container my-fn with an input ConfigMap containing `data: {foo: bar}`
$ kpt fn eval DIR --image gcr.io/example.com/my-fn:v1.0.0 -- foo=bar
```

```
# execute executable my-fn on the resources in DIR directory and
# write output back to DIR
$ kpt fn eval DIR --exec-path ./my-fn
```

```
# execute container my-fn on the resources in DIR directory,
# save structured results in /tmp/my-results dir and write output back to DIR
$ kpt fn eval DIR --image gcr.io/example.com/my-fn --results-dir /tmp/my-results-dir
```

```
# execute container my-fn on the resources in DIR directory with network access enabled,
# and write output back to DIR
$ kpt fn eval DIR --image gcr.io/example.com/my-fn --network 
```

```
# execute container my-fn on the resource in DIR and export KUBECONFIG
# and foo environment variable
$ kpt fn eval DIR --image gcr.io/example.com/my-fn --env KUBECONFIG -e foo=bar
```

```
# execute kubeval function by mounting schema from a local directory on wordpress package
$ kpt fn eval --image gcr.io/kpt-fn/kubeval:v0.1 \
  --mount type=bind,src="/path/to/schema-dir",dst=/schema-dir \
  --as-current-user wordpress -- additional_schema_locations=/schema-dir
```

```
# chaining functions using the unix pipe to set namespace and set labels on
# wordpress package
$ kpt fn source wordpress \
  | kpt fn eval --image gcr.io/kpt-fn/set-namespace:v0.1 - -- namespace=mywordpress \
  | kpt fn eval --image gcr.io/kpt-fn/set-labels:v0.1 - -- label_name=color label_value=orange \
  | kpt fn sink wordpress
```
<!--mdtogo-->

[docker volumes]: https://docs.docker.com/storage/volumes/
[Imperative Function Execution]: /book/04-using-functions/02-imperative-function-execution
[Function Specification]: /book/05-developing-functions/02-function-specification