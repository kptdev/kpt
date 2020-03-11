## create-setter

Create or modify a field setter

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="/kpt/gifs/cfg-create-setter.cast" speed="1" theme="solarized-dark" cols="100" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

    kpt tutorial cfg create-setter

[tutorial-script]

### Synopsis

    kpt cfg create-setter DIR NAME VALUE

  DIR

    A directory containing Resource configuration.
    e.g. hello-world/

  NAME

    The name of the substitution to create.  This is both the name that will be given
    to the *set* command, and that will be referenced by fields.
    e.g. replicas

  VALUE

    The new value of the setter.
    e.g. 3

#### Field Setters

Field setters are OpenAPI definitions that define how fields may be modified programmatically
using the *set* command.  The OpenAPI definitions for setters are defined in a Kptfile
and referenced by fields which they set through an OpenAPI reference as a line comment
(e.g. # {"$ref":"#/definitions/..."}).

Setters may be manually created by editing the Kptfile, or programmatically created using the
`create-setter` command.  The `create-setter` command will 1) create a new OpenAPI definition
for a setter in the Kptfile, and 2) identify all fields matching the setter value and create
an OpenAPI reference to the setter for each.

    # create or update a setter named replicas
    kpt cfg create-setter hello-world/ replicas 3

Example setter definition in a Kptfile:

```yaml
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      x-k8s-cli:
        setter:
          name: "replicas"
          value: "3"
```

This setter is named "replicas" and can be provided to the *set* command to change
all fields which reference it to the setter's value.

Example setter referenced from a field in a configuration file:

```yaml
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 3  # {"$ref":"#/definitions/io.k8s.cli.setters.replicas"}
```

Setters may have types specified which ensure that the configuration is always serialized
correctly as yaml 1.1 -- e.g. if a string field such as an annotation or arg has the value
"on", then it would need to be quoted otherwise it will be parsed as a bool by yaml 1.1.

A type may be specified using the --type flag, and accepts string,integer,boolean as values.
The resulting OpenAPI definition looks like:

    # create or update a setter named version which sets the "version" annotation
    kpt cfg create-setter hello-world/ version 3 --field "annotations.version" --type string

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

And the configuration looks like:

```yaml
kind: Deployment
metadata:
  name: foo
  annotations:
    version: "3" # {"$ref":"#/definitions/io.k8s.cli.setters.version"}
```

Setters may be configured to accept enumeration values which map to different values set
on the fields.  For example setting cpu resources to small, medium, large -- and mapping
these to specific cpu values.  This may be done by manually modifying the Kptfile openAPI
definitions as shown here:

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

And the configuration looks like:

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

    # create a setter called replicas for fields matching "3"
    kpt cfg create-setter DIR/ replicas 3

    # scope creating setter references to a specified field
    kpt cfg create-setter DIR/ replicas 3 --field "replicas"

    # scope creating setter references to a specified field path
    kpt cfg create-setter DIR/ replicas 3 --field "spec.replicas"

    # create a setter called replicas with a description and set-by
    kpt cfg create-setter DIR/ replicas 3 --set-by "package-default" \
        --description "good starter value"

    # scope create a setter with a type.  the setter will make sure the set fields
    # always parse as strings with a yaml 1.1 parser (e.g. values such as 1,on,true wil
    # be quoted so they are parsed as strings)
    # only the final part of the the field path is specified
    kpt cfg create-setter DIR/ app nginx --field "annotations.app" --type string

### 

[tutorial-script]: ../gifs/cfg-create-setter.sh
