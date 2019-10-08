## kpt reconcile wrap

Wrap a reconcile command in xargs and pipe to merge + fmt

### Synopsis

Wrap a reconcile command in xargs and pipe to merge + fmt.

Porcelain for running CMD wrapped in 'kpt xargs' and piping the result to
'kpt merge | kpt fmt --set-filenames'.

If KPT_OVERRIDE_PKG is set to a directory in the container, wrap will also read
the contents of the override package directory and merge them on top of the CMD
output.


```
kpt reconcile wrap CMD... [flags]
```

### Examples

```



```

### Options

```
      --env-only              only set env vars, not arguments. (default true)
  -h, --help                  help for wrap
      --wrap-kind string      wrap the input xargs give to the command in this type. (default "List")
      --wrap-version string   wrap the input xargs give to the command in this type. (default "v1")
```

### SEE ALSO

* [kpt reconcile](kpt_reconcile.md)	 - Reconcile runs transformers against the package Resources

