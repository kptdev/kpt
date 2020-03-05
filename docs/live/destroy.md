## destroy

remove a package from the cluster

### Synopsis

    kpt live destroy DIRECTORY [flags]

The destroy command removes all files belonging to a package from
the cluster.

Args:
  DIRECTORY:
    One directory that contain k8s manifests. The directory
    must contain exactly one ConfigMap with the grouping object annotation.
