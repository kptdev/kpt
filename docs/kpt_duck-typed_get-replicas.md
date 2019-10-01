## kpt duck-typed get-replicas

Get the replicas for a Resource

### Synopsis

Get the container image for a workload

Args:

  NAME:
    Name of the Resource and Container from which to get the image.

Command is enabled for a package by having a Resource with the field: spec.template.spec.containers


```
kpt duck-typed get-replicas NAME [flags]
```

### Examples

```
kpt  get replicas NAME
```

### Options

```
  -h, --help   help for get-replicas
```

### SEE ALSO

* [kpt duck-typed](kpt_duck-typed.md)	 - Duck-typed commands are enabled for packages based off the package's content

