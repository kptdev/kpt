---
title: "Display local package contents"
linkTitle: "Display"
weight: 2
type: docs
description: >
    Display the contents of a local package using kpt cfg for rendering.
---

{{% hide %}}

<!-- @makeWorkplace @verifyGuides-->
```
# Set up workspace for the test.
TEST_HOME=$(mktemp -d)
cd $TEST_HOME
```

{{% /hide %}}

*Tools can parse and render configuration so it is easier for humans to read.*

## Topics

[kpt cfg count], [kpt cfg tree],
[kpt cfg grep], [kpt cfg cat]

## Steps

1. [Fetch a remote package](#fetch-a-remote-package)
2. [Summarize resource counts](#summarize-resource-counts)
3. [Display resources as a tree](#display-resources-as-a-tree)
4. [Filter resources](#filter-resources)
5. [Dump all resources](#dump-all-resources)

## Fetch a remote package

Packages are fetched from remote git repository subdirectories with
[kpt pkg get].  In this guide we will use the [kubernetes examples] repository
as a public package catalogue.

### Fetch Command

<!-- @fetchPackage @verifyGuides-->
```sh
kpt pkg get https://github.com/kubernetes/examples/staging/ examples
```

Fetch the entire examples/staging directory as a kpt package under `examples`.
This will contain many resources.

### List Command

```sh
tree examples
```

### List Output

```sh
examples/
├── Kptfile
├── cloud-controller-manager
│   └── persistent-volume-label-initializer-config.yaml
├── cluster-dns
│   ├── README.md
...

79 directories, 329 files
```

The package is composed of 79 directories and, 329 files.  This is too many
to work with using tools such as `less`.

## Summarize resource counts

### Count Example Command 1

<!-- @countExamples @verifyGuides-->
```sh
kpt cfg count examples/
```

The [`kpt cfg count`][kpt cfg count] command summarizes the resource counts to
show the shape of a package.

### Count Example Output 1

```sh
...
Deployment: 10
Endpoints: 1
InitializerConfiguration: 1
Namespace: 4
Pod: 45
...
```

### Count Example Command 2

<!-- @countCockroachdb @verifyGuides-->
```sh
kpt cfg count examples/cockroachdb/
```

Running [`count`][kpt cfg count] on a subdirectory will summarize that
directory even if it doesn't have a Kptfile.

### Count Example Output 2

```sh
PodDisruptionBudget: 1
Service: 2
StatefulSet: 1
```

### Count Example Command 3

<!-- @countAll @verifyGuides-->
```sh
kpt cfg count examples/ --kind=false
```

The total aggregate resource count can be shown with `--kind=false`

### Count Example Output 3

```sh
201
```

## Display resources as a tree

### Display Command

<!-- @treeCockroachdb @verifyGuides-->
```sh
kpt cfg tree examples/cockroachdb/ --image --replicas
```

Because the raw YAML configuration may be difficult for humans to easily
view and understand, kpt provides a command for rendering configuration
as a tree.  Flags may be provided to print specific fields under the resources.

### Display Output

```sh
examples/cockroachdb
├── [cockroachdb-statefulset.yaml]  Service cockroachdb
├── [cockroachdb-statefulset.yaml]  StatefulSet cockroachdb
│   ├── spec.replicas: 3
│   └── spec.template.spec.containers
│       └── 0
│           └── image: cockroachdb/cockroach:v1.1.0
├── [cockroachdb-statefulset.yaml]  PodDisruptionBudget cockroachdb-budget
└── [cockroachdb-statefulset.yaml]  Service cockroachdb-public
```

In addition to the built-in printable fields, [`kpt cfg tree`][kpt cfg tree]
will print arbitrary fields by providing the `--field` flag.

## Filter resources

### Filter Command

<!-- @filterExamples @verifyGuides-->
```sh
kpt cfg grep "spec.replicas>3" examples | kpt cfg tree --replicas
```

Grep can be used to filter resources by field values.  The output of
[`kpt cfg grep`][kpt cfg grep] is the matching full resource configuration, which
may be piped to tree for rendering.

### Filter Output

```sh
.
├── storage/minio
│   └── [minio-distributed-statefulset.yaml]  StatefulSet minio
│       └── spec.replicas: 4
├── sysdig-cloud
│   └── [sysdig-rc.yaml]  ReplicationController sysdig-agent
│       └── spec.replicas: 100
└── volumes/vsphere
    └── [simple-statefulset.yaml]  StatefulSet web
        └── spec.replicas: 14
```

## Dump all resources

### Dump Command

<!-- @catCockroachdb @verifyGuides-->
```sh
kpt cfg cat examples/cockroachdb
# Temporary workaround for https://github.com/GoogleContainerTools/kpt/issues/1050
echo "\n"
```

The raw YAML configuration may be dumped using [`kpt cfg cat`][kpt cfg cat].
This will print only the YAML for Kubernetes resources.

### Dump Output

```sh
apiVersion: v1
kind: Service
metadata:
  # This service is meant to be used by clients of the database. It exposes a
  # ClusterIP that will automatically load balance connections to the different
  # database pods.
  name: cockroachdb-public
  labels:
    app: cockroachdb
...
```

[kubernetes examples]: https://github.com/kubernetes/examples
[kpt cfg count]: ../../../reference/cfg/count/
[kpt cfg tree]: ../../../reference/cfg/tree/
[kpt cfg grep]: ../../../reference/cfg/grep/
[kpt cfg cat]: ../../../reference/cfg/cat/
[kpt pkg get]: ../../../reference/pkg/get/
