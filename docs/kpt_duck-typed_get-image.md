## kpt duck-typed get-image

Get image for a container

### Synopsis

Get image for a container

Args:

  NAME:
    Name of the Resource and Container from which to get the image.

Command is enabled for a package by having a Resource with the field: spec.template.spec.containers


```
kpt duck-typed get-image NAME [flags]
```

### Examples

```
kpt  get image NAME
```

### Options

```
  -h, --help   help for get-image
```

### SEE ALSO

* [kpt duck-typed](kpt_duck-typed.md)	 - Duck-typed commands are enabled for packages based off the package's content

