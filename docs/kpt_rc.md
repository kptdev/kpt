## kpt rc

Count Resources Config from a local package

### Synopsis

Count Resources Config from a local package.

  DIR:
    Path to local package directory.


```
kpt rc DIR... [flags]
```

### Examples

```
# print Resource counts from a package
kpt rc my-package/

```

### Options

```
  -h, --help                  help for rc
      --include-subpackages   also print resources from subpackages. (default true)
      --kind                  count resources by kind. (default true)
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

