## kpt reconcile

Reconcile runs transformers against the package Resources

### Synopsis

Reconcile runs transformers against the package Resources.

  DIR:
    Path to local package directory.

See 'kpt help apis transformers' for more information.


```
kpt reconcile DIR/ [flags]
```

### Examples

```
# reconcile package transformers
kpt reconcile my-package/

```

### Options

```
      --api-resource strings   additional API resources to reconcile
      --dry-run                print results to stdout
  -h, --help                   help for reconcile
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

