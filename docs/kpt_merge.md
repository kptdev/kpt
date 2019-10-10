## kpt merge

Merge Resource configuration files

### Synopsis

Merge Resource configuration files

Merge reads Kubernetes Resource yaml configuration files from stdin or sources packages and write
the result to stdout or a destination package.

Resources are merged using the Resource [apiVersion, kind, name, namespace] as the key.  If any of
these are missing, merge will default the missing values to empty.

Resources specified later are high-precedence (the source) and Resources specified
earlier are lower-precedence (the destination).

Merge uses the following rules for merging a source Resource into a destination Resource:

See: 'kpt help apis merge' for more information on merge rules.


```
kpt merge [SOURCE_DIR...] [DESTINATION_DIR] [flags]
```

### Examples

```
cat resources_and_patches.yaml | kpt merge > merged_resources.yaml
```

### Options

```
  -h, --help           help for merge
      --invert-order   if true, merge Resources in the reverse order
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

