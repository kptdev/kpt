## kpt duck-typed set-replicas

Set the replicas for a Resource

### Synopsis

Set the container image on a workload

Args:

  NAME:
    Name of the Resource and Container on which to set the image.

Command is enabled for a package by having a Resource with the field: spec.template.spec.containers


```
kpt duck-typed set-replicas NAME [flags]
```

### Examples

```
kpt  set replicas NAME --value VALUE
```

### Options

```
  -h, --help        help for set-replicas
      --value int   the new replicas value
```

### SEE ALSO

* [kpt duck-typed](kpt_duck-typed.md)	 - Duck-typed commands are enabled for packages based off the package's content

