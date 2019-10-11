## kpt tree

Display package Resource structure

### Synopsis

Display package Resource structure.

  DIR:
    Path to local package directory.


```
kpt tree DIR [flags]
```

### Examples

```
# print package structure
kpt tree my-package/

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
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

