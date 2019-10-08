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

# wrap Resource config from a package in an ResourceList
kpt cat my-package/ --wrap-kind ResourceList --wrap-version kpt.dev/v1alpha1 --function-config fn.yaml

# unwrap Resource config from a package in an ResourceList
... | kpt cat

# write as json
kpt cat my-package

```

### Options

```
      --annotate                 annotate resources with their file origins.
      --format                   format resource config yaml before printing. (default true)
      --function-config string   path to function config to put in ResourceList -- only if wrapped in a ResourceList.
  -h, --help                     help for cat
      --include-subpackages      also print resources from subpackages. (default true)
      --strip-comments           remove comments from yaml.
      --style strings            yaml styles to apply.  may be 'TaggedStyle', 'DoubleQuotedStyle', 'LiteralStyle', 'FoldedStyle', 'FlowStyle'.
      --wrap-kind string         if set, wrap the output in this list type kind.
      --wrap-version string      if set, wrap the output in this list type apiVersion.
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

