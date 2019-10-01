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

- Map fields specified in both the source and destination are merged recursively.
- Scalar fields specified in both the source and destination have the destination value replaced
  by the source value.
- Lists elements specified in both the source and destination are merged:
  - As a scalar if the list elements do not have an associative key.
  - As maps if the lists do have an associative key -- the associative key is used as the map key
  - The following are associative in precedence order:
    "mountPath", "devicePath", "ip", "type", "topologyKey", "name", "containerPort"
- Any fields specified only in the destination are kept in the destination as is.
- Any fields specified only in the source are copied to the destination.
- Fields specified in the sources as null will be cleared from the destination if present
- Comments are merged on all fields and list elements from the source if they are specified,
  on the source, otherwise the destination comments are kept as is.


```
kpt merge [SOURCE_DIR...] [DESTINATION_DIR] [flags]
```

### Examples

```
cat resources_and_patches.yaml | kpt merge > merged_resources.yaml
```

### Options

```
  -h, --help   help for merge
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

