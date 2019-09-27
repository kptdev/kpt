## kpt duck-typed get-memory-limits

Get memory-limits for a container

### Synopsis

Get memory-limits for a container.

Args:

  NAME:
    Name of the Resource and Container from which to get memory-limits.

Command is enabled for a package by having a Resource with the field: spec.template.spec.containers


```
kpt duck-typed get-memory-limits NAME [flags]
```

### Examples

```
kpt  get memory-limits NAME
```

### Options

```
  -h, --help   help for get-memory-limits
```

### SEE ALSO

* [kpt duck-typed](kpt_duck-typed.md)	 - Duck-typed commands are enabled for packages based off the package's content

