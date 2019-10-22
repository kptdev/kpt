## kpt tree

Display package Resource structure

### Synopsis

Display package Resource structure.

kpt tree may be used to print Resources in a package or cluster, preserving structure

Args:

  DIR:
    Path to local package directory.

Resource fields may be printed as part of the Resources by specifying the fields as flags.

kpt tree has build-in support for printing common fields, such as replicas, container images,
container names, etc.

kpt tree supports printing arbitrary fields using the '--field' flag.

By default, kpt tree uses the package structure for the tree structure, however when printing
from the cluster, the owners structure may be used instead.


```
kpt tree DIR [flags]
```

### Examples

```
# print Resources using package structure
kpt tree my-package/

# print replicas, container name, and container image and fields for Resources
kpt tree my-package --replicas --image --name

# print all common Resource fields
kpt tree my-package/ --all

# print the "foo"" annotation
kpt tree my-package/ --field "metadata.annotations.foo" 

# print the "foo"" annotation
kubectl get all -o yaml | kpt tree my-package/ --structure=graph \
  --field="status.conditions[type=Completed].status"

# print live Resources from a cluster using owners for structure
kubectl get all -o yaml | kpt tree --replicas --name --image --structure=graph


# print live Resources using owners for structure
kubectl get all,applications,releasetracks -o yaml | kpt tree --structure=graph \
  --name --image --replicas \
  --field="status.conditions[type=Completed].status" \
  --field="status.conditions[type=Complete].status" \
  --field="status.conditions[type=Ready].status" \
  --field="status.conditions[type=ContainersReady].status"

```

### Options

```
      --all                       print all field infos
      --args                      print args field
      --command                   print command field
      --env                       print env field
      --exclude-non-reconcilers   if true, exclude non-reconciler Resources in the output.
      --field strings             print field
  -h, --help                      help for tree
      --image                     print image field
      --include-reconcilers       if true, include reconciler Resources in the output.
      --include-subpackages       also print resources from subpackages. (default true)
      --name                      print name field
      --ports                     print ports field
      --replicas                  print replicas field
      --resources                 print resources field
      --structure string          structure to use for the tree.  may be 'package' or 'owners'. (default "package")
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

