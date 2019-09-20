## kpt man

Format and display package documentation if it exists

### Synopsis

Format and display package documentation if it exists.
Args:

  LOCAL_PKG_DIR:
    path to locally fetched package.

  If package documentation is missing or 'man' is not installed, the command will fail.

```
kpt man LOCAL_PKG_DIR [flags]
```

### Examples

```
  # display package documentation
  kpt man my-package/

  # display subpackage documentation
  kpt man my-package/sub-package/
```

### Options

```
  -h, --help   help for man
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

