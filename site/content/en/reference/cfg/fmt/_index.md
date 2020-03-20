---
title: "Fmt"
linkTitle: "fmt"
weight: 4
type: docs
description: >
   Format configuration files
---
<!--mdtogo:Short
    Format configuration files
-->

{{< asciinema key="cfg-fmt" rows="10" preload="1" >}}

Format formats the field ordering in YAML configuration files.

Inputs may be directories, files or STDIN.  Formatted resources must
include both `apiVersion` and `kind` fields.

- Stdin inputs are formatted and written to stdout
- File inputs (args) are formatted and written back to the file
- Directory inputs (args) are walked, each encountered .yaml and .yml file
  acts as an input

For inputs which contain multiple yaml documents separated by \n---\n,
each document will be formatted and written back to the file in the original
order.

Field ordering roughly follows the ordering defined in the source Kubernetes
resource definitions (i.e. go structures), falling back on lexicographical
sorting for unrecognized fields.

Unordered list item ordering is defined for specific Resource types and
field paths.

- .spec.template.spec.containers (by element name)
- .webhooks.rules.operations (by element value)

### Examples
<!--mdtogo:Examples-->
```sh
# format file1.yaml and file2.yml
kpt cfg fmt file1.yaml file2.yml
```

```sh
# format all *.yaml and *.yml recursively traversing directories
kpt cfg fmt my-dir/
```

```sh
# format kubectl output
kubectl get -o yaml deployments | kpt cfg fmt
```

```sh
# format kustomize output
kustomize build | kpt cfg fmt
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
    kpt cfg fmt [DIR]

    DIR:
      Path to a package directory.  Reads from STDIN if not provided.
<!--mdtogo-->
