## kpt duck-typed get-cpu-limits

Get cpu-limits for a container

### Synopsis

Get cpu-limits for a container.

Args:

  NAME:
    Name of the Resource and Container from which to get cpu-limits.

Command is enabled for a package by having a Resource with the field: spec.template.spec.containers


```
kpt duck-typed get-cpu-limits NAME [flags]
```

### Examples

```
kpt  get cpu-limits NAME
```

### Options

```
  -h, --help   help for get-cpu-limits
```

### SEE ALSO

* [kpt duck-typed](kpt_duck-typed.md)	 - Duck-typed commands are enabled for packages based off the package's content

