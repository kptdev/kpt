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
kpt functionality is subdivided into command groups, each of which operates on
a particular set of entities, with a consistent command syntax and pattern of
inputs and outputs.
<!--mdtogo-->

### Examples
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

By default, kpt will first try to fetch the OpenAPI schema from the cluster 
given by the current context. If that doesn't work, it will silently fall back 
to using the builtin schema.

```
--k8s-schema-source
  Set the source for the OpenAPI schema. Allowed values are cluster, file, or
  builtin. If an OpenAPI schema can't be find at the given source, kpt will 
  return an error.

--k8s-schema-path
  The path to an OpenAPI schema file. The default value is ./openapi.json
```

| Command Group | Description                                                                     |  Reads From     | Writes To       |
|---------------|---------------------------------------------------------------------------------|-----------------|-----------------|
| [pkg]         | fetch, update, and sync configuration files using git                           | remote git      | local directory |
| [cfg]         | examine and modify configuration files                                          | local directory | local directory |
| [fn]          | generate, transform, validate configuration files using containerized functions | local directory | local directory |
| [live]        | reconcile the live state with configuration files                               | local directory | remote cluster  |

[updating]: pkg/update
[functions]: fn/run
[setters]: cfg/set
[gcr.io/kpt-dev/kpt]: https://gcr.io/kpt-dev/kpt
[pkg]: pkg/
[cfg]: cfg/
[fn]: fn/
[live]: live/
