## kpt duck-typed get-memory-requests

Get memory-requests for a container

### Synopsis

Get memory-requests for a container.

Args:

  NAME:
    Name of the Resource and Container from which to get memory-requests.

Command is enabled for a package by having a Resource with the field: spec.template.spec.containers


```
kpt duck-typed get-memory-requests NAME [flags]
```

### Examples

```
kpt  get memory-requests NAME
```

### Options

```
  -h, --help   help for get-memory-requests
```

### SEE ALSO

* [kpt duck-typed](kpt_duck-typed.md)	 - Duck-typed commands are enabled for packages based off the package's content

