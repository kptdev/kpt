## kpt duck-typed get-cpu-requests

Get cpu-requests for a container

### Synopsis

Get cpu-requests for a container.

Args:

  NAME:
    Name of the Resource and Container from which to get cpu-requests.

Command is enabled for a package by having a Resource with the field: spec.template.spec.containers


```
kpt duck-typed get-cpu-requests NAME [flags]
```

### Examples

```
kpt  get cpu-requests NAME
```

### Options

```
  -h, --help   help for get-cpu-requests
```

### SEE ALSO

* [kpt duck-typed](kpt_duck-typed.md)	 - Duck-typed commands are enabled for packages based off the package's content

