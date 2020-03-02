## preview

preview shows the changes apply will make against the live state of the cluster

### Synopsis

    kpt live preview [FILENAME... | DIRECTORY] [flags]

The preview command will run through the same steps as apply, but 
it will only print what would happen when running apply against the current
live cluster state. 

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
    
Flags:
  no-prune:
    If true, previously applied objects will not be pruned.