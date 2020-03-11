---
title: "Create-setter"
linkTitle: "create-setter"
weight: 4
type: docs
description: >
   Create a setter for one or more field
---

{{< asciinema key="cfg-create-setter" rows="10" preload="1" >}}

Setters provide a solution for template-free setting or substitution of field
values through package metadata (OpenAPI).  They are a safer alternative to
other substitution techniques which do not have the context of the
structured data -- e.g. using `sed` to replace values.

The OpenAPI definitions for setters are defined in a Kptfile and referenced by
a fields through comments on the fields.

Setters may be invoked to programmatically modify the configuration
using `kpt cfg set` to set and/or substitute values.

#### Creating a Setter

Setters may be created either manually (by editing the Kptfile directly), or
programmatically (through the `create-setter` command).  The `create-setter`
command will:

1. create a new OpenAPI definition for a setter in the Kptfile
2. create references to the setter definition on the resource fields

```yaml
# Kptfile -- original
openAPI:
  definitions: {}
```

```yaml
# deployment.yaml -- original
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 3
```

```sh
# create or update a setter named "replicas"
# match fields with the value "3"
kpt cfg create-setter hello-world/ replicas 3
```

```yaml
# Kptfile -- updated
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      x-k8s-cli:
        setter:
          name: "replicas"
          value: "3"
```

```yaml
# deployment.yaml -- updated
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 3 # {"$ref":"#/definitions/io.k8s.cli.setters.replicas"}
```

#### Invoking a Setter

```yaml
# deployment.yaml -- original
kind: Deployment
metadata:
 name: helloworld-gke
 labels:
   app: hello
spec:
 replicas: 3 # {"$ref":"#/definitions/io.k8s.cli.setters.replicas"}
```

```sh
# set the replicas field to 5
kpt cfg set DIR/ replicas 5
```

```yaml
# deployment.yaml -- updated
kind: Deployment
metadata:
 name: helloworld-gke
 labels:
   app: hello
spec:
 replicas: 5 # {"$ref":"#/definitions/io.k8s.cli.setters.replicas"}
```

#### Types

Setters may have types specified which ensure that the configuration is always
serialized correctly as yaml 1.1 -- e.g. if a string field such as an
annotation or arg has the value "on", then it would need to be quoted otherwise
it will be parsed as a bool by yaml 1.1.

This may be done by modifying the Kptfile OpenAPI definitions as shown here:

```yaml
openAPI:
  definitions:
    io.k8s.cli.setters.version:
      x-k8s-cli:
        setter:
          name: "version"
          value: "3"
      type: string
```

Set would change the configuration like this:

```yaml
kind: Deployment
metadata:
  name: foo
  annotations:
    version: "3" # {"$ref":"#/definitions/io.k8s.cli.setters.version"}
```

#### Enumerations

Setters may be configured to map an enum input to a different value set
in the configuration.

e.g. users set `small`, `medium`, `large` cpu sizes, and these are mapped
to numeric values set in the configuration.

This may be done by modifying the Kptfile OpenAPI definitions as shown here:

```yaml
openAPI:
  definitions:
    io.k8s.cli.setters.cpu:
      x-k8s-cli:
        setter:
          name: "cpu"
          value: "small"
          # enumValues will replace the user provided key with the
          # map value when setting fields.
          enumValues:
            small: "0.5"
            medium: "2"
            large: "4"
```

Set would change the configuration like this:

```yaml
kind: Deployment
metadata:
  name: foo
spec:
  template:
    spec:
      containers:
      - name: foo
    resources:
      requests:
        cpu: "0.5" # {"$ref":"#/definitions/io.k8s.cli.setters.cpu"}
```

### Examples

```sh
# create a setter called replicas for fields matching "3"
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

### Synopsis

    kpt cfg create-setter DIR NAME VALUE

    DIR:
      Path to a package directory

    NAME:
      The name of the substitution to create.  This is both the name that will
      be given to the *set* command, and that will be referenced by fields.
      e.g. replicas

    VALUE
      The new value of the setter.
      e.g. 3
