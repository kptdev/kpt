## kpt duck-typed set-cpu-limits

Set cpu-limits for a container

### Synopsis

Set cpu-limits for a container.

Args:

  NAME:
    Name of the Resource and Container on which to set cpu-limits.

Command is enabled for a package by having a Resource with the field: spec.template.spec.containers


```
kpt duck-typed set-cpu-limits NAME [flags]
```

### Examples

```
kpt  set cpu-limits NAME --value VALUE
```

### Options

```
  -h, --help           help for set-cpu-limits
      --value string   the new value
```

### SEE ALSO

* [kpt duck-typed](kpt_duck-typed.md)	 - Duck-typed commands are enabled for packages based off the package's content

