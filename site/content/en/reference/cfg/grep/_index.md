---
title: "Grep"
linkTitle: "grep"
weight: 4
type: docs
description: >
  Filter resources by their field values
---

<!--mdtogo:Short
    Filter resources by their field values
-->

{{< asciinema key="cfg-grep" rows="10" preload="1" >}}

Grep reads resources from a package or stdin and filters them by their
field values.

Grep may have sources such as `kubectl get -o yaml` piped to it, or may
be piped to other commands such as `kpt cfg tree` for display.

### Examples

<!--mdtogo:Examples-->

```sh
# find Deployment Resources
kpt cfg grep "kind=Deployment" my-dir/
```

```sh
# find Resources named nginx
kpt cfg grep "metadata.name=nginx" my-dir/
```

```sh
# use tree to display matching Resources
kpt cfg grep "metadata.name=nginx" my-dir/ | kpt cfg tree
```

```sh
# look for Resources matching a specific container image
kpt cfg grep "spec.template.spec.containers[name=nginx].image=nginx:1\.7\.9" \
    my-dir/ | kpt cfg tree
```

<!--mdtogo-->

### Synopsis

<!--mdtogo:Long-->

```
kpt cfg grep QUERY DIR
```

#### Args

```
    QUERY:
      Query to match expressed as 'path.to.field=value'.
      Maps and fields are matched as '.field-name' or '.map-key'
      List elements are matched as '[list-elem-field=field-value]'
      The value to match is expressed as '=value'
      '.' as part of a key or value can be escaped as '\.'

    DIR:
      Path to a package directory
```

<!--mdtogo-->

#### Flags

```sh
--annotate
  annotate resources with their file origins. (default true)

--invert-match, -v
  keep resources NOT matching the specified pattern

--recurse-subpackages, -R
  Grep recursively in all the nested subpackages
```
