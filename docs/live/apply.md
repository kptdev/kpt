## apply

apply a package to the cluster

### Synopsis

    kpt live apply DIRECTORY [flags]

The apply command creates, updates or deletes any resources in the cluster to
make the state of resources in the cluster match the desired state as specified
through the set of manifests. This command is similar to the apply command
available in kubectl, but also has support for pruning and waiting until all
resources has been fully reconciled.

Args:
  DIRECTORY:
    One directory that contain k8s manifests. The directory
    must contain exactly one ConfigMap with the grouping object annotation.
    
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

Applied resource set:
  This refers to the set of resources in the directory applied to cluster as a group.
  `kpt live apply` tracks the state of your applied resource set and related configuration. This
  helps `kpt` to reliably reconcile the real world resources with your configuration changes.

Prune:
  `kpt live apply` can declaratively delete the resources which are not part of your
  applied resource set anymore. `kubectl apply` also has a similar functionality with --prune
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
