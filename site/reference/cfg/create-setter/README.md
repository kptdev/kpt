---
title: "Create-setter"
linkTitle: "create-setter"
weight: 4
type: docs
description: >
  Create a setter for one or more field
---

<!--mdtogo:Short
    Create a setter for one or more field
-->

{{< asciinema key="cfg-create-setter" rows="10" preload="1" >}}

Setters provide a solution for template-free setting or substitution of field
values through package metadata (OpenAPI). They are a safer alternative to
other substitution techniques which do not have the context of the
structured data -- e.g. using `sed` to replace values.

See the [creating setters] guide for more info on creating setters.

### Examples

<!--mdtogo:Examples-->

```sh
# create a setter called replicas for fields matching value "3"
kpt cfg create-setter DIR/ replicas 3
```

```sh
# scope creating setter references to a specified field
kpt cfg create-setter DIR/ replicas 3 --field "replicas"
```

```sh
# scope creating setter references to a specified field path
kpt cfg create-setter DIR/ replicas 3 --field "spec.replicas"
```

```sh
# create a setter called replicas with a description and set-by
kpt cfg create-setter DIR/ replicas 3 --set-by "package-default" \
    --description "good starter value"
```

```sh
# scope create a setter with a type.  the setter will make sure the set fields
# always parse as strings with a yaml 1.1 parser (e.g. values such as 1,on,true
# will be quoted so they are parsed as strings)
# only the final part of the the field path is specified
kpt cfg create-setter DIR/ app nginx --field "annotations.app" --type string
```

<!--mdtogo-->

### Synopsis

<!--mdtogo:Long-->

```sh
kpt cfg create-setter DIR NAME VALUE

DIR:
  Path to a package directory

NAME:
  The name of the setter to create.  This is both the name that will
  be given to the *set* command, and that will be referenced by fields.
  e.g. replicas

VALUE
  The value of the filed for which setter reference must be added.
  e.g. 3
```

<!--mdtogo-->

#### Flags

```sh
--description string
  record a description for the current setter value.

--field string
  name of the field to set, a suffix of the path to the field, or the full path
  to the field. Default is to match all fields.

--value
  Optional flag, alternative to specifying the value as an argument
  e.g. used to specify values that start with '-'

--recurse-subpackages, -R
  create setter recursively in all the nested subpackages

--required
  indicates that this setter must be set by package consumer before live apply/preview

--schema-path string
  openAPI schema file path for setter constraints -- file content
  e.g. {"type": "string", "maxLength": 15, "enum": ["allowedValue1", "allowedValue2"]}

--set-by string
  record who the field was default by.

--type string
  OpenAPI field type for the setter -- e.g. integer,boolean,string.

--value string
  alternative to specifying the value as an argument. e.g. used to specify values
  that start with '-'
```

[creating setters]: ../../../guides/producer/setters/
