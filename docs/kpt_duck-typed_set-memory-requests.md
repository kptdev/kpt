## kpt duck-typed set-memory-requests

Set memory-requests for a container

### Synopsis

Set memory-requests for a container.

Args:

  NAME:
    Name of the Resource and Container on which to set memory-requests.

Command is enabled for a package by having a Resource with the field: spec.template.spec.containers


```
kpt duck-typed set-memory-requests NAME [flags]
```

### Examples

```
kpt  set memory-requests NAME --value VALUE
```

### Options

```
  -h, --help           help for set-memory-requests
      --value string   the new value
```

### SEE ALSO

* [kpt duck-typed](kpt_duck-typed.md)	 - Duck-typed commands are enabled for packages based off the package's content

