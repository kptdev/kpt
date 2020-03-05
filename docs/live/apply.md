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
