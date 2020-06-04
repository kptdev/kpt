---
title: "Display local package contents"
linkTitle: "Display"
weight: 2
type: docs
description: >
    Display the contents of a local package using kpt cfg for rendering.
---

*Tools can parse and render configuration so it is easier for humans to read.*

## Topics

[kpt cfg count](/reference/pkg/count), [kpt cfg tree](/reference/pkg/get),
[kpt cfg grep](/reference/pkg/count), [kpt cfg cat](/reference/pkg/count)

## Steps

1. [Fetch a remote package](#fetch-a-remote-package)
2. [Summarize resource counts](#summarize-resource-counts)
3. [Display resources as a tree](#display-resources-as-a-tree)
4. [Filter resources](#filter-resources)
5. [Dump all resources](#dump-all-resources)

## Fetch a remote package

Packages are fetched from remote git repository subdirectories with
[kpt pkg get](/reference/pkg/get).  In this guide we will use the [kubernetes examples] repository
as a public package catalogue.

##### Command

```sh
kpt pkg get https://github.com/kubernetes/examples/staging/ examples
```

Fetch the entire examples/staging directory as a kpt package under `examples`.
This will contain many resources.

##### Command

```sh
tree examples
```

##### Output

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

##### Command

```sh
kpt cfg count examples/
```

The `kpt cfg count` command summarizes the resource counts to show the shape of a
package.

##### Output

```sh
...
Deployment: 10
Endpoints: 1
InitializerConfiguration: 1
Namespace: 4
Pod: 45
...
```

##### Command

```sh
kpt cfg count examples/cockroachdb/
```

Running `count` on a subdirectory will summarize that directory even if
it doesn't have a Kptfile.

##### Output

```sh
PodDisruptionBudget: 1
Service: 2
StatefulSet: 1
```


##### Command

```sh
kpt cfg count examples/ --kind=false
```

The total aggregate resource count can be shown with `--kind=false`

##### Output

```sh
201
```

## Display resources as a tree

##### Command

```sh
kpt cfg tree examples/cockroachdb/ --image --replicas 
```

Because the raw YAML configuration may be difficult for humans to easily
view and understand, kpt provides a command for rendering configuration
as a tree.  Flags may be provided to print specific fields under the resources.

##### Output

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

In addition to the built-in printable fields, `kpt cfg tree` will print
arbitrary fields by providing the `--field` flag.

## Filter resources

##### Command

```sh
kpt cfg grep "spec.replicas>3" examples | kpt cfg tree --replicas
```

Grep can be used to filter resources by field values.  The output of
`kpt cfg grep` is the matching full resource configuration, which
may be piped to tree for rendering.

##### Output

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

##### Command

```sh
kpt cfg cat examples/cockroachdb
```

The raw YAML configuration may be dumped using `kpt cfg cat`.  This will
print only the YAML for Kubernetes resources.

##### Output

```sh
apiVersion: v1
kind: Service
metadata:
  # This service is meant to be used by clients of the database. It exposes a ClusterIP that will
  # automatically load balance connections to the different database pods.
  name: cockroachdb-public
  labels:
    app: cockroachdb
...
```

[kubernetes examples]: https://github.com/kubernetes/examples
[kpt cfg count]: ../../../reference/pkg/count
[kpt cfg tree]: ../../../reference/pkg/get
[kpt cfg grep]: ../../../reference/pkg/count
[kpt cfg cat]: ../../../reference/pkg/count
