## preview

preview shows the changes apply will make against the live state of the cluster

### Synopsis

    kpt live preview DIRECTORY [flags]

The preview command will run through the same steps as apply, but 
it will only print what would happen when running apply against the current
live cluster state. 

Args:
  DIRECTORY:
    One directory that contain k8s manifests. The directory
    must contain exactly one ConfigMap with the grouping object annotation.
    
Flags:
  destroy:
    If true, dry-run deletion of all resources.
