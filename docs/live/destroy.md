## destroy

remove a package from the cluster

### Synopsis

    kpt live destroy [FILENAME... | DIRECTORY] [flags]

The destroy command removes all files belonging to a package from
the cluster.

Args:
  NONE:
    Input will be read from StdIn. Exactly one ConfigMap manifest
    with the grouping object annotation must be present.

  FILENAME:
    A set of files that contains k8s manifests. Exactly one of them
    needs to be a ConfigMap with the grouping object annotation.
    
  DIRECTORY:
    One or more directories that contain k8s manifests. The directories 
    must contain exactly one ConfigMap with the grouping object annotation.
