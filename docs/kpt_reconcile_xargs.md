## kpt reconcile xargs

Convert functionConfig to commandline flags

### Synopsis

Convert functionConfig to commandline flags.

xargs reads a ResourceList from stdin and parses the functionConfig field.  xargs then
reads each of the fields under .spec and parses them as flags.  If the fields have non-scalar
values, then xargs encoded the values as yaml strings.

  CMD:
    The command to run and pass the functionConfig as arguments.


```
kpt reconcile xargs -- CMD... [flags]
```

### Examples

```

# given this example functionConfig in config.yaml
kind: Foo
spec:
  flag1: value1
  flag2: value2
items:
- 2
- 1

# this command:
$ kpt cat pkg/ --function-config config.yaml --wrap-kind ResourceList | kpt reconcile xargs -- app

# is equivalent to this command:
$ kpt cat pkg/ --function-config config.yaml --wrap-kind ResourceList | app --flag1=value1 --flag2=value2 2 1

# echo: prints the app arguments
$ kpt cat pkg/ --function-config config.yaml --wrap-kind ResourceList | kpt reconcile xargs -- echo
--flag1=value1 --flag2=value2 2 1

# env: prints the app env
$ kpt cat pkg/ --function-config config.yaml --wrap-kind ResourceList | kpt reconcile xargs -- env

# cat: prints the app stdin -- prints the package contents and functionConfig wrapped in a
# ResourceList
$ kpt cat pkg/ --function-config config.yaml --wrap-kind ResourceList | kpt reconcile xargs --no-flags -- env


```

### Options

```
      --env-only              only add env vars, not flags
  -h, --help                  help for xargs
      --wrap-kind string      wrap the input xargs give to the command in this type. (default "List")
      --wrap-version string   wrap the input xargs give to the command in this type. (default "v1")
```

### SEE ALSO

* [kpt reconcile](kpt_reconcile.md)	 - Reconcile runs transformers against the package Resources

