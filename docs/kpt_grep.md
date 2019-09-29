## kpt grep

Search for matching Resources in a package

### Synopsis

Search for matching Resources in a package.
  QUERY:
    Query to match expressed as 'path.to.field=value'.
    Maps and fields are matched as '.field-name' or '.map-key'
    List elements are matched as '[list-elem-field=field-value]'
    The value to match is expressed as '=value'
    '.' as part of a key or value can be escaped as '\.'

  DIR:
    Path to local package directory.


```
kpt grep QUERY [DIR]... [flags]
```

### Examples

```
# find Deployment Resources
kpt grep "kind=Deployment" my-package/

# find Resources named nginx
kpt grep "metadata.name=nginx" my-package/

# use tree to display matching Resources
kpt grep "metadata.name=nginx" my-package/ | kpt tree

# look for Resources matching a specific container image
kpt grep "spec.template.spec.containers[name=nginx].image=nginx:1\.7\.9" my-package/ | kpt tree

```

### Options

```
      --annotate              annotate resources with their file origins. (default true)
  -h, --help                  help for grep
      --include-subpackages   also print resources from subpackages. (default true)
  -v, --invert-match           Selected Resources are those not matching any of the specified patterns..
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

