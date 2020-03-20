---
title: "Apply"
linkTitle: "apply"
type: docs
description: >
   Apply a package to the cluster (create, update, delete)
---
<!--mdtogo:Short
    Apply a package to the cluster (create, update, delete)
-->

{{< asciinema key="live-apply" rows="10" preload="1" >}}

Apply creates, updates and deletes resources in the cluster to make the remote
cluster resources match the local package configuration.

Kpt apply is and extended version of kubectl apply, with added support
for pruning and blocking on resource status.

Kpt apply has a different usage pattern (args + flags) from kubectl to make
it consistent with other kpt commands.

#### `kubectl apply` vs `kpt live apply`

|                     | `kubectl apply`            | `kpt live apply`          |
|---------------------|----------------------------|---------------------------|
|Usage                | kubectl apply -f /dir      | kpt live apply /dir       |
|Applied resource set | Not tracked                | Tracked                   |
|Prune                | Imperative and error prone | Declarative and reliable  |
|Status               | Not supported              | Supported                 |

##### Applied resource set

This refers to the set of resources in the directory applied to cluster as a
group.  `kpt live apply` tracks the state of your applied resource set and
related configuration. This helps `kpt` to reliably reconcile the real world
resources with your configuration changes.

### Prune

kpt live apply will automatically delete resources which have been
previously applied, but which are no longer included. This clean-up
functionality is called pruning. For example, consider a package
which has been applied with the following three resources:

```
service-1 (Service)
deployment-1 (Deployment)
config-map-1 (ConfigMap)
```

Then imagine the package is updated to contain the following resources,
including a new ConfigMap named `config-map-2` (Notice that `config-map-1`
is not part of the updated package):

```
service-1 (Service)
deployment-1 (Deployment)
config-map-2 (ConfigMap)
```

When the updated package is applied, `config-map-1` is automatically
deleted (pruned) since it is omitted.


In order to take advantage of this automatic clean-up, a package must contain
a **grouping object template**, which is a ConfigMap with a special label. An example is:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-grouping-object
  labels:
    cli-utils.sigs.k8s.io/inventory-id: test-group
```

And the special label is:

```
cli-utils.sigs.k8s.io/inventory-id: *group-name*
```

`kpt live apply` recognizes this template from the special label, and based
on this kpt will create new grouping object with the metadata of all applied
objects in the ConfigMap's data field. Subsequent `kpt live apply` commands can
then query the grouping object, and calculate the omitted objects, cleaning up
accordingly. When a grouping object is created in the cluster, a hash suffix
is added to the name. Example:

```
test-grouping-object-17b4dba8
```

### Status (wait-for-reconcile)

kpt live apply also has support for computing status for resources. This is 
useful during apply for making sure that not only are the set of resources applied
into the cluster, but also that the desired state expressed in the resource are
fully reconciled in the cluster. An example of this could be applying a deployment. Without
looking at the status, the operation would be reported as successful as soon as the
deployment resource has been created in the apiserver. With status, kpt live apply will
wait until the desired number of pods have been created and become available.

Status is computed through a set of rules for specific types, and
functionality for polling a set of resources and computing the aggregate status
for the set. For CRDs, there is a set of recommendations that if followed, will allow
kpt live apply to correctly compute status.

### Synopsis
<!--mdtogo:Long-->
    kpt live apply DIR [flags]

#### Args

    DIR:
      Path to a package directory.  The directory must contain exactly
      one ConfigMap with the grouping object annotation.

#### Flags:

    --wait-for-reconcile:
      If true, after all resources have been applied, the cluster will
      be polled until either all resources have been fully reconciled
      or the timeout is reached.

    --wait-polling-period:
      The frequency with which the cluster will be polled to determine 
      the status of the applied resources. The default value is every 2 seconds.

    --wait-timeout:
      The threshold for how long to wait for all resources to reconcile before
      giving up. The default value is 1 minute.
<!--mdtogo-->
