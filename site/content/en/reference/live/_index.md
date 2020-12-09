---
title: "Live"
linkTitle: "live"
weight: 3
type: docs
description: >
   Reconcile configuration files with the live state
---
<!--mdtogo:Short
    Reconcile configuration files with the live state
-->

{{< asciinema key="live" rows="10" preload="1" >}}

<!--mdtogo:Long-->
| Reads From              | Writes To                |
|-------------------------|--------------------------|
| local files             | cluster                  |
| cluster                 | stdout                   |

Live contains the next-generation versions of apply related commands for
deploying local configuration packages to a cluster.

### How to Upgrade to the Next Generation Inventory Object

#### What is an Inventory Object

An inventory object is the automatically generated object which keeps track
of the set of objects applied together. The current inventory object type
is a [**ConfigMap**](https://kubernetes.io/docs/concepts/configuration/configmap/),
and it is usually defined in a package file called
**inventory-template.yaml**. This file is created from an invocation of
`kpt live init`. A typical use of the inventory object is to prune (delete)
objects omitted locally.

#### What is a Next Generation Inventory Object

The next generation inventory object is a **ResourceGroup** custom resource
replacing the current [**ConfigMap**](https://kubernetes.io/docs/concepts/configuration/configmap/).
Because the new inventory object is a
custom resource, you must have permissions to add a custom resource
definition (CRD) to the cluster. The individual command to add the
**ResourceGroup** CRD is:

```sh
export RESOURCE_GROUP_INVENTORY=1
kpt live install-resource-group
```

#### Upgrade Scenario 1: New (Uninitialized) Packages

Packages which are newly downloaded and uninitialized should follow the
following steps:

1. `export RESOURCE_GROUP_INVENTORY=1`
2. `kpt live init <PACKAGE DIR>`

The `init` step adds an `inventory` section in the
package Kptfile. The package is now eligible to be applied
with `kpt live apply <PACKAGE DIR>`. This command will
automatically apply the **ResourceGroup** CRD if has not
already been applied.

#### Upgrade Scenario 2: Existing (Initialized) Packages

Existing packages which have already been initialized, should follow
the following steps:

1. `export RESOURCE_GROUP_INVENTORY=1`
2. `kpt live migrate <PACKAGE DIR>`

Initially, the `migrate` command applies the **ResourceGroup** CRD.
Then the `migrate` command replaces the
[**ConfigMap**](https://kubernetes.io/docs/concepts/configuration/configmap/)
inventory object in the cluster (if it exists) with a **ResourceGroup**
custom resource. The `migrate` command also deletes the local inventory
[**ConfigMap**](https://kubernetes.io/docs/concepts/configuration/configmap/)
config file (usually **inventory-template.yaml**). If this local
[**ConfigMap**](https://kubernetes.io/docs/concepts/configuration/configmap/)
file is stored in a github repository, the removal
needs to be committed to the repository to finalize the removal.
Finally, the `migrate` command adds an `inventory`
section to the Kptfile if it did not already exist. Updates to
the package can now be applied using `kpt live apply <PACKAGE DIR>`.

#### Troubleshooting and Verifying

* Error: unable to apply **ResourceGroup** CRD

```sh
$ kpt live install-resource-group
error: unable to add resourcegroups.kpt.dev
```

This message means the user does not have permissions to add the
**ResourceGroup** CRD to the cluster. Until the user is able to
find someone with sufficient privileges to apply the CRD, the
user can continue to use the previous
[**ConfigMap**](https://kubernetes.io/docs/concepts/configuration/configmap/)
inventory object by unsetting the environment variable:

```sh
unset RESOURCE_GROUP_INVENTORY
```

* Error: configuration already created initialization error

```sh
$ kpt live init <PACKAGE DIR>
error: ResourceGroup configuration has already been created. Changing
them after a package has been applied to the cluster can lead to
undesired results. Use the --force flag to suppress this error.
```

This message means the **ResourceGroup** initialization has
*already* happened. Unless, you want to `--force` new values,
this can safely be ignored.

* How to check if the **ResourceGroup** CRD has *not* been
successfully applied to the cluster

```sh
$ kubectl get resourcegroups.kpt.dev
error: the server doesn't have a resource type "resourcegroups"
```

If a `No resources found` message is returned, the
**ResourceGroup** CRD *has* been successfully applied,
but there are no **ResourceGroup** custom resources
found in the namespace. Example:

```sh
$ kubectl get resourcegroups.kpt.dev
No resources found in default namespace.
```

* How to check if the applied inventory object in the cluster has
been upgraded to a **ResourceGroup** custom resource

```sh
$ kubectl get resourcegroups.kpt.dev -n <PKG NAMESPACE> --selector='cli-utils.sigs.k8s.io/inventory-id' -o name
resourcegroup.kpt.dev/inventory-62308923
```

* How to check if the applied inventory object in the cluster is
not upgraded and is still a **ConfigMap**

```sh
$ kubectl get cm -n <PKG NAMESPACE> --selector='cli-utils.sigs.k8s.io/inventory-id' -o name
configmap/inventory-62308923
```

<!--mdtogo-->
