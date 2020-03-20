---
title: "Cfg"
linkTitle: "cfg"
type: docs
weight: 2
description: >
    Display and modify JSON or YAML configuration
---
<!--mdtogo:Short
    Display and modify JSON or YAML configuration
-->

{{< asciinema key="cfg" rows="10" preload="1" >}}

<!--mdtogo:Long-->
| Reads From              | Writes To                |
|-------------------------|--------------------------|
| local files or stdin    | local files or stdout    |

The `cfg` command group contains subcommands which read and write
local YAML files.  They are focused on providing porcelain on top
of workflows which would otherwise require viewing and editing YAML
directly.

Many cfg subcommands may also read from STDIN, allowing them to be paired
with other tools such as `kubectl get`.
<!--mdtogo-->

    kpt cfg [SUBCOMMAND]

### Examples
<!--mdtogo:Examples-->
```sh
# print the package using tree based structure
$ kpt cfg tree helloworld --name --image --replicas
helloworld
├── [deploy.yaml]  Deployment helloworld-gke
│   ├── spec.replicas: 5
│   └── spec.template.spec.containers
│       └── 0
│           ├── name: helloworld-gke
│           └── image: gcr.io/kpt-dev/helloworld-gke:0.1.0
└── [service.yaml]  Service helloworld-gke
```

```sh
#  list available setters
$ kpt cfg list-setters helloworld replicas
    NAME          DESCRIPTION        VALUE    TYPE     COUNT   SETBY
  replicas   'helloworld replicas'   5       integer   1

# set a high-level knob
$ kpt cfg set helloworld replicas 3
set 1 fields
```
<!--mdtogo-->
