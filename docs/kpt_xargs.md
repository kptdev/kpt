## kpt xargs

Convert functionConfig to commandline flags

### Synopsis

Convert functionConfig to commandline flags.

xargs reads a ResourceList from stdin and parses the functionConfig field.  xargs then
reads each of the fields under .spec and parses them as flags.  If the fields have non-scalar
values, then xargs encoded the values as yaml strings.

  CMD:
    The command to run and pass the functionConfig as arguments.


```
kpt xargs CMD... [flags]
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
$ kpt cat pkg/ --function-config config.yaml --wrap-kind ResourceList | kpt xargs app

# is equivalent to this command:
$ kpt cat pkg/ --function-config config.yaml --wrap-kind ResourceList | app --flag1=value1 --flag2=value2 2 1

# echo: prints the app arguments
$ kpt cat pkg/ --function-config config.yaml --wrap-kind ResourceList | kpt xargs echo
--flag1=value1 --flag2=value2 2 1

# cat: prints the app stdin -- prints the package contents and functionConfig wrapped in a
# ResourceList
$ kpt cat pkg/ --function-config config.yaml --wrap-kind ResourceList | kpt xargs cat


```

### Options

```
  -h, --help   help for xargs
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

