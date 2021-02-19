---
title: "Command Reference"
linkTitle: "Command Reference"
type: docs
weight: 40
menu:
  main:
    weight: 3
description: >
    Overview of kpt commands
---

<!--mdtogo:Short
    Overview of kpt commands
-->

{{< asciinema key="kpt" rows="10" preload="1" >}}

<!--mdtogo:Long-->

kpt functionality is subdivided into the following command groups, each of
which operates on a particular set of entities, with a consistent command
syntax and pattern of inputs and outputs.

| Command Group | Description                                                                     | Reads From      | Writes To       |
| ------------- | ------------------------------------------------------------------------------- | --------------- | --------------- |
| [pkg]         | fetch, update, and sync configuration files using git                           | remote git      | local directory |
| [cfg]         | examine and modify configuration files                                          | local directory | local directory |
| [fn]          | generate, transform, validate configuration files using containerized functions | local directory | local directory |
| [live]        | reconcile the live state with configuration files                               | local directory | remote cluster  |

<!--mdtogo-->

### Examples

The following are examples of running each kpt command group.

<!--mdtogo:Examples-->

```sh
# get a package
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.5.0 helloworld
fetching package /package-examples/helloworld-set from \
  https://github.com/GoogleContainerTools/kpt to helloworld
```

```sh
# list setters and set a value
$ kpt cfg list-setters helloworld
NAME            DESCRIPTION         VALUE    TYPE     COUNT   SETBY
http-port   'helloworld port'         80      integer   3
image-tag   'hello-world image tag'   v0.3.0  string    1
replicas    'helloworld replicas'     5       integer   1

$ kpt cfg set helloworld replicas 3 --set-by pwittrock  --description 'reason'
set 1 fields
```

```sh
# get a package and run a validation function
kpt pkg get https://github.com/GoogleContainerTools/kpt-functions-sdk.git/example-configs example-configs
mkdir results/
kpt fn run example-configs/ --results-dir results/ --image gcr.io/kpt-functions/validate-rolebinding:results -- subject_name=bob@foo-corp.com
```

```sh
# apply the package to a cluster
$ kpt live apply --reconcile-timeout=10m helloworld
...
all resources has reached the Current status
```

<!--mdtogo-->

### OpenAPI schema

Kpt relies on the OpenAPI schema for Kubernetes to understand the structure
of kubernetes manifests. Kpt already comes with a builtin
OpenAPI schema, but that will obviously not include any CRDs. So in some
situations it might be beneficial to use a schema that accurately reflects both
the correct version of Kubernetes and the CRDs used. Kpt provides a few global
flags to allows users to specify the schema that should be used.

By default, kpt will use the builtin schema.

```sh
--k8s-schema-source
  Set the source for the OpenAPI schema. Allowed values are cluster, file, or
  builtin. If an OpenAPI schema can't be find at the given source, kpt will
  return an error.

--k8s-schema-path
  The path to an OpenAPI schema file. The default value is ./openapi.json
```

### Global flags

Kpt exposes many global flags in addition to the ones listed above to allow
customization of how kpt works. This is primarily around logging and how kpt
connects to your kubernetes cluster. Some flags that are unlikely to be useful
to most users are hidden in the cli, but this section lists all flags accepted
by kpt.

```
--add_dir_header
  If true, adds the file directory to the header
--alsologtostderr
  log to standard error as well as files
--as string
  Username to impersonate for the operation
--as-group stringArray
  Group to impersonate for the operation, this flag can be repeated to
  specify multiple groups.
--cache-dir string
  Default HTTP cache directory (default "/Users/<user>/.kube/http-cache")
--certificate-authority string
  Path to a cert file for the certificate authority
--client-certificate string
  Path to a client certificate file for TLS
--client-key string
  Path to a client key file for TLS
--cluster string
  The name of the kubeconfig cluster to use
--context string
  The name of the kubeconfig context to use
-h, --help
  Help for kpt
--insecure-skip-tls-verify
  If true, the servers certificate will not be checked for validity.
  This will make your HTTPS connections insecure
--install-completion
  Install shell completion
--k8s-schema-path string
  Path to the kubernetes openAPI schema file (default "./openapi.json")
--k8s-schema-source string
  Source for the kubernetes openAPI schema (default "builtin")
--kubeconfig string
  Path to the kubeconfig file to use for CLI requests.
--log-flush-frequency duration
  Maximum number of seconds between log flushes (default 5s)
--log_backtrace_at traceLocation
  When logging hits line file:N, emit a stack trace (default :0)
--log_dir string
  If non-empty, write log files in this directory
--log_file string
  If non-empty, use this log file
--log_file_max_size uint
  Defines the maximum size a log file can grow to. Unit is megabytes.
  If the value is 0, the maximum file size is unlimited. (default 1800)
--logtostderr
  Log to standard error instead of files (default true)
--match-server-version
  Require server version to match client version
-n, --namespace string
  If present, the namespace scope for this CLI request
--password string
  Password for basic authentication to the API server
--request-timeout string
  The length of time to wait before giving up on a single server request.
  Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h).
  A value of zero means don't timeout requests. (default "0")
-s, --server string
  The address and port of the Kubernetes API server
--skip_headers
  If true, avoid header prefixes in the log messages
--skip_log_headers
  If true, avoid headers when opening log files
--stack-trace
  Print a stack-trace on failure
--stderrthreshold severity
  Logs at or above this threshold go to stderr (default 2)
--token string
  Bearer token for authentication to the API server
--user string
  The name of the kubeconfig user to use
--username string
  Username for basic authentication to the API server
-v, --v Level
  Number for the log level verbosity
--vmodule moduleSpec
  Comma-separated list of pattern=N settings for file-filtered logging
```

### Next Steps

- Learn about kpt [architecture] including major influences and a high-level
  comparison with kustomize.
- Read kpt [guides] for how to produce and consume packages and integrate with
  a wider ecosystem of tools.
- Consult the [FAQ] for answers to common questions.

[pkg]: pkg/
[cfg]: cfg/
[fn]: fn/
[live]: live/
[architecture]: ../concepts/architecture/
[guides]: ../guides/
[FAQ]: ../faq/
