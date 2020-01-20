## tree

Display Resource structure from a directory or stdin.

![alt text][tutorial]

    kpt tutorial cfg tree

[tutorial-script]

### Synopsis

kpt cfg tree may be used to print Resources in a directory or cluster, preserving structure

Args:

  DIR:
    Path to local directory directory.

Resource fields may be printed as part of the Resources by specifying the fields as flags.

kpt cfg tree has build-in support for printing common fields, such as replicas, container images,
container names, etc.

kpt cfg tree supports printing arbitrary fields using the '--field' flag.

By default, kpt cfg tree uses Resource graph structure if any relationships between resources (ownerReferences)
are detected, as is typically the case when printing from a cluster. Otherwise, directory graph structure is used. The
graph structure can also be selected explicitly using the '--graph-structure' flag.

### Examples

    # print Resources using directory structure
    kpt cfg tree my-dir/

    # print replicas, container name, and container image and fields for Resources
    kpt cfg tree my-dir --replicas --image --name

    # print all common Resource fields
    kpt cfg tree my-dir/ --all

    # print the "foo"" annotation
    kpt cfg tree my-dir/ --field "metadata.annotations.foo"

    # print the "foo"" annotation
    kubectl get all -o yaml | kpt cfg tree \
      --field="status.conditions[type=Completed].status"

    # print live Resources from a cluster using owners for graph structure
    kubectl get all -o yaml | kpt cfg tree --replicas --name --image

    # print live Resources with status condition fields
    kubectl get all -o yaml | kpt cfg tree \
      --name --image --replicas \
      --field="status.conditions[type=Completed].status" \
      --field="status.conditions[type=Complete].status" \
      --field="status.conditions[type=Ready].status" \
      --field="status.conditions[type=ContainersReady].status"

###

[tutorial]: https://storage.googleapis.com/kpt-dev/docs/cfg-tree.gif "kpt cfg tree"
[tutorial-script]: ../../gifs/cfg-tree.sh
