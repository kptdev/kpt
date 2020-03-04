## apply

apply a package to the cluster

### Synopsis

    kpt live apply [DIRECTORY] [flags]

The apply command creates, updates or deletes any resources in the cluster to
make the state of resources in the cluster match the desired state as specified
through the set of manifests. This command is similar to the apply command
available in kubectl, but also has support for pruning and waiting until all
resources has been fully reconciled.

Args:

  DIRECTORY:
    Directory that contain k8s manifests with no sub-folders.
    
Flags:
    
  wait-for-reconcile:
    If true, after all resources have been applied, the cluster will
    be polled until either all resources have been fully reconciled
    or the timeout is reached.
    
  wait-polling-period:
    The frequency with which the cluster will be polled to determine 
    the status of the applied resources. The default value is every 2 seconds.
    
  wait-timeout:
    The threshold for how long to wait for all resources to reconcile before
    giving up. The default value is 1 minute.

#### `kubectl apply` vs `kpt live apply`

|                     | `kubectl apply`            | `kpt live apply`          |
|---------------------|----------------------------|---------------------------|
|Usage                | kubectl apply -f /dir      | kpt live apply /dir       |
|Applied resource set | Not tracked                | Tracked                   |
|Prune                | Imperative and error prone | Declarative and reliable  |
|wait-for-reconcile   | Not supported              | Supported                 |

Usage:
  In terms of usage, both `kubectl apply` and `kpt live apply` follow similar pattern.
  The user experience remains unchanged.

Applied resource set(ARS):
  `kpt live apply` tracks the state of your applied resource set and related configuration. This
  helps `kpt` to reliably reconcile the real world ARS with your configuration changes.

Prune:
  `kpt live apply` can declaratively delete the resources which are not part of your
  ARS anymore. `kubectl apply` also has a similar functionality with --prune
  flag. However, it heavily depends on labels provided by user, which is imperative and
  error prone. On the other hand, prune is default option for `kpt live apply` and app
  state is completely managed and tracked by kpt. The only additional step users should
  perform in order to enjoy the benefits of prune is to add the grouping object ConfigMap
  to your package with UNIQUE label.

wait-for-reconcile:
  `kubectl apply` will simply apply the resources. Users must write their own logic
  to poll using `kubectl get` to check if the applied resources were fully reconciled.
  `kpt live apply` can wait till the resources are fully reconciled with continuous status
  updates.

#### Prune Example 
Consider an example of a directory with three resources which should be applied to
the cluster. This is a simple use case and both `kpt live apply` and `kubectl apply`
work in the same way.

```
service-1 (Service)
deployment-1 (Deployment)
config-map-1 (ConfigMap)
```

So far so good, but imagine the package is updated to contain the following resources,
including a new ConfigMap named `config-map-2` (Notice that `config-map-1`
is not part of the updated package). This means the user intent is to delete `config-map-1`
and create `config-map-2`

```
service-1 (Service)
deployment-1 (Deployment)
config-map-2 (ConfigMap)
```

At this point, it gets tricky with `kubectl apply` to achieve the task. As kubectl doesn't
store the information that `config-map-1` belongs to this applied resource set, it just creates
`config-map-2` and doesn't delete `config-map-1`. Alternative options with kubectl to prune
resources is to use --prune flag specifying label or by prune-whitelist which are very dangerous
as it might delete other resources not related to this resource set sharing same cluster. On the 
other hand, `kpt live apply` tracks the state of Applied resource set and when the updated package
is applied, `config-map-1` is automatically deleted (pruned).

##### Inventory ConfigMap
For tracking the applied resource set state, `kpt live init` generates an inventory ConfigMap
in input directory with special unique label provided by the user. The input directory is considered
as the boundary for this label with k8s resources group. This label is used to generate state tracking
ConfigMap in the cluster while applying the resources. This label must be unique to the directory.
If not, it might lead to accidental deletions of other resource sets sharing the same cluster.

#### Status
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

###