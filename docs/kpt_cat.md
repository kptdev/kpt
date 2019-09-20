## kpt cat

Print Resource Config from a local package

### Synopsis

Print Resource Config from a local package.

  DIR:
    Path to local package directory.


```
kpt cat DIR... [flags]
```

### Examples

```
# print Resource config from a package
kpt cat my-package/

```

### Options

```
      --annotate              annotate resources with their file origins. (default true)
      --format                format resource config yaml before printing. (default true)
  -h, --help                  help for cat
      --include-subpackages   also print resources from subpackages. (default true)
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

