## fetch

Display the current status of the Resources

### Synopsis

Fetches the state of all provided resources from the cluster and displays the status in
a table.

The list of resources are provided as manifests either on the filesystem or on StdIn. 

  DIR:
    Path to local directory.

### Examples

    # Read resources from the filesystem and wait up to 1 minute for all of them to become Current
    kpt svr status fetch my-dir/

    # Fetch all resources in the cluster and wait up to 5 minutes for all of them to become Current
    kubectl get all --all-namespaces -o yaml | resource status fetch
