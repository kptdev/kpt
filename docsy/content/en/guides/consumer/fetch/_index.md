---
title: "Fetch a remote package"
linkTitle: "Fetch"
weight: 1
type: docs
description: >
    Fetch a package from a remote git repository and apply its contents to
    a cluster
---

*Any git directory containing configuration files may be used by kpt
as a package.*

## Topics

[kpt pkg get], [Kptfile]

## Steps

1. [Fetch a remote package](#fetch-a-remote-package)
2. [View the Kptfile](#view-the-kptfile)
3. [View the package contents](#view-the-package-contents)
4. [Apply the package to a cluster](#apply-the-package-to-a-cluster)
5. [View the applied package](#view-the-applied-package)

## Fetch a remote package

Packages are fetched from remote git repository subdirectories with
[kpt pkg get].  In this guide we will use the [kubernetes examples] repository
as a public package catalogue.

##### Command

```sh
kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb cockroachdb
```

##### Output

```
fetching package staging/cockroachdb from https://github.com/kubernetes/examples to cockroachdb
```

This command copied the contents of the `staging/cockroachdb` subdirectory
in the `https://github.com/kubernetes/examples` to the local folder
`cockroachdb`.

{{% pageinfo color="info" %}}
- any git subdirectory containing configuration may be fetched as a package
- the local package directory name does NOT need to match the remote
  directory name
- including `.git` as part of the repo name is optional for well known hosts
  such as GitHub
{{% /pageinfo %}}

## View the Kptfile

##### Command

The upstream commit and branch / tag reference are stored in the
package's [Kptfile] these can be used to update the
package later.

```sh
cat cockroachdb/Kptfile
```

Print the `Kptfile` written by `kpt pkg get` to see the upstream package data.

##### Output

```
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
    name: cockroachdb
upstream:
    type: git
    git:
        commit: 629c9459a9f25e468cce8af28350a03e62c5f67d
        repo: https://github.com/kubernetes/examples
        directory: staging/cockroachdb
        ref: master
```

## View the package contents

The primary package artifacts are Kubernetes [resource configuration]
(e.g. YAML files), however packages may also include supporting
artifacts such as documentation.

##### Command

```sh
tree cockroachdb/
```

##### Output

```sh
cockroachdb/
├── Kptfile
├── OWNERS
├── README.md
├── cockroachdb-statefulset.yaml
├── demo.sh
└── minikube.sh

0 directories, 6 files
```

The cockroachdb package fetched from [kubernetes examples] contains a
`cockroachdb-statefulset.yaml` file with the resource configuration, as well
as other files included in the directory.  The Kptfile was created by
`kpt pkg get` for storing package state.  If the upstream package already
defines a Kptfile, `kpt pkg get` will update the Kptfile copied from upstream
rather than replacing it.

The package contains 2 resource configuration files -- `deploy.yaml` and
`service.yaml`.  These are the same types of resource configuration that
would be applied with `kubectl apply`

##### Command

```sh
head cockroachdb/cockroachdb-statefulset.yaml
```

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
spec:
  ports:
```

In this package, the `cockroachdb/cockroachdb-statefulset.yaml` is plain old
resource configuration -- nothing special here.  Other packages may have
resource configuration which is marked up with metadata as comments that
kpt can use to provide high-level commands for reading and writing the
configuration.

## Apply the package to a cluster

Use `kubectl apply` to deploy the local package to a remote cluster.

##### Command

```sh
kubectl apply -R -f cockroachdb
```

##### Output

```sh
service/cockroachdb-public created
service/cockroachdb created
poddisruptionbudget.policy/cockroachdb-budget unchanged
statefulset.apps/cockroachdb created
```

{{% pageinfo color="info" %}}
This guide showed using `kubectl apply` to apply in order to demonstrate how kpt
packages work out of the box with existing tools.

Kp also provides a next-generation set of apply commands under the [kpt live]
command.
{{% /pageinfo %}}


## View the applied package

Once applied to the cluster, the remote resources can be displayed using
the normal tools such as `kubectl get`.

##### Command

```sh
kubectl get all
```

##### Output

```sh
NAME                READY   STATUS    RESTARTS   AGE
pod/cockroachdb-0   1/1     Running   0          54s
pod/cockroachdb-1   1/1     Running   0          41s
pod/cockroachdb-2   1/1     Running   0          27s

NAME                         TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)              AGE
service/cockroachdb          ClusterIP   None         <none>        26257/TCP,8080/TCP   55s
service/cockroachdb-public   ClusterIP   10.48.2.5    <none>        26257/TCP,8080/TCP   55s
service/kubernetes           ClusterIP   10.48.0.1    <none>        443/TCP              26m

NAME                           READY   AGE
statefulset.apps/cockroachdb   3/3     54s
```


[kubernetes examples]: https://github.com/kubernetes/examples
[resource configuration]: https://kubernetes.io/docs/concepts/configuration/overview/#general-configuration-tips
[kpt pkg get]: ../../..//reference/pkg/get
[Kptfile]: ../../../api-reference/kptfile
[kpt live]: ../../../reference/live
