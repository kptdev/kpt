## kpt duck-typed set-image

Set the image on a container

### Synopsis

Set the image on a container.

Args:

  NAME:
    Name of the Resource and Container on which to set the image.

Command is enabled for a package by having a Resource with the field: spec.template.spec.containers


```
kpt duck-typed set-image NAME [flags]
```

### Examples

```
kpt  set image NAME --value VALUE
```

### Options

```
  -h, --help           help for set-image
      --value string   the new image value
```

### SEE ALSO

* [kpt duck-typed](kpt_duck-typed.md)	 - Duck-typed commands are enabled for packages based off the package's content

