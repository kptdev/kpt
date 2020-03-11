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

{{< asciinema key="kpt" rows="10" preload="1" >}}

kpt functionality is subdivided into command groups, each of which operates on
a particular set of entities, with a consistent command syntax and pattern of
inputs and outputs.

### Examples

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
# apply the package to a clsuter
$ kpt live apply --wait-for-reconcile helloworld
...
all resources has reached the Current status
```

| Command Group | Description                                                                     |  Reads From     | Writes To       |
|---------------|---------------------------------------------------------------------------------|-----------------|-----------------|
| [pkg]         | fetch, update, and sync configuration files using git                           | remote git      | local directory |
| [cfg]         | examine and modify configuration files                                          | local directory | local directory |
| [fn]          | generate, transform, validate configuration files using containerized functions | local directory | local directory |
| [live]        | reconcile the live state with configuration files                               | local directory | remote cluster  |


[updating]: pkg/updat
[functions]: fn/run
[setters]: cfg/set
[gcr.io/kpt-dev/kpt]: https://gcr.io/kpt-dev/kpt
[pkg]: pkg/
[cfg]: cfg/
[fn]: fn/
[live]: live/
