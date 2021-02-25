---
title: "Setters"
linkTitle: "Setters"
weight: 2
type: docs
description: >
  Create high-level [setters] to provide imperative configuration editing
  commands.
---

{{% hide %}}

<!-- @makeWorkplace @verifyGuides-->

```
# Set up workspace for the test.
TEST_HOME=$(mktemp -d)
cd $TEST_HOME
```

{{% /hide %}}

{{% pageinfo color="primary" %}}
Creating a setter requires that the package has a Kptfile. If one does
not exist for the package, run `kpt pkg init DIR/` to create one.
{{% /pageinfo %}}

{{% hide %}}

<!-- @createKptfile @verifyGuides-->

```
kpt pkg init .
```

{{% /hide %}}

## Setters explained

Following is a short explanation of the command that will be demonstrated
in this guide.

### Data model

- Fields reference setters through OpenAPI definitions specified as
  line comments -- e.g. `# { "$kpt-set": "replicas-setter" }`
- OpenAPI definitions are provided through the Kptfile

### Command control flow

1. Read the package Kptfile and resources.
2. Change the setter OpenAPI value in the Kptfile
3. Locate all fields which reference the setter and change their values.
4. Write both the modified Kptfile and resources back to the package.

![img](/static/images/set-command.svg)

This guide walks you through an end-to-end example to create, invoke and delete setters.

## Steps

1. [Creating a Setter](#creating-a-setter)
2. [Targeting fields using the path](#targeting-fields-using-the-path)
3. [Invoking a Setter](#invoking-a-setter)
4. [Deleting a Setter](#deleting-a-setter)
5. [OpenAPI Validations](#openapi-validations)
6. [Required Setters](#required-setters)
7. [Setting Lists](#setting-lists)

#### Creating a Setter

Setters may be created either manually (by editing the Kptfile directly), or
programmatically (through the `create-setter` command). The `create-setter`
command will:

1. create a new OpenAPI definition for a setter in the Kptfile
2. create references to the setter definition on the resource fields

<!-- @createResource @verifyGuides-->

```sh
cat <<EOF >deployment.yaml
# deployment.yaml -- original
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 3
EOF
```

<!-- @createSetter @verifyGuides-->

```sh
# create or update a setter named "replicas"
# match fields with the value "3"
kpt cfg create-setter . replicas 3
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
  replicas: 3 # {"$kpt-set":"replicas"}
```

{{% hide %}}

<!-- @validateCreateSetter @verifyGuides-->

```
grep "io.k8s.cli.setters.replicas" Kptfile
grep 'value: "3"' Kptfile
grep 'replicas: 3 # {"$kpt-set":"replicas"}' deployment.yaml
```

{{% /hide %}}

#### Targeting fields using the path

The basic way to create a setter only matches fields based on the value. But
in some situations there might be several fields with the same value, but not all
of them should be targeted by the setter. In these situations, use the `--field`
flag to provide either the name of the field, the full path to the field, or a
partial (suffix) path to the field. Only fields that match both the path and
the value will be targeted by the setter.

```sh
# create a setter named "replicas" and but only target the field name replicas in the spec
kpt cfg create-setter hello-world/ replicas 3 --field="spec.replicas"
```

```yaml
# deployment-foo.yaml
kind: Deployment
metadata:
  name: foo
  annotations:
    replicas: 3
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
```

The path is always just the names of the properties on the path to the field,
regardless whether the field is nested inside a sequence. Targeting specific
elements inside a sequence is not supported.

```sh
# create a setter named "mountName" and only target the name of the volume mount
kpt cfg create-setter hello-world/ mountName nginx --field="spec.containers.volumeMounts.name"
```

```yaml
# deployment-nginx.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  containers:
    - name: nginx
      image: nginx
      volumeMounts:
        - name: nginx # {"$kpt-set":"mountName"}
          mountPath: /usr/share/nginx
        - name: temp
          mountPath: /tmp
```

#### Invoking a Setter

```yaml
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
```

<!-- @setReplicas @verifyGuides-->

```sh
# set the replicas field to 5
kpt cfg set . replicas 5
```

```yaml
# deployment.yaml -- updated
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
```

{{% hide %}}

<!-- @validateSetSetter @verifyGuides-->

```
grep 'value: "5"' Kptfile
grep 'replicas: 5 # {"$kpt-set":"replicas"}' deployment.yaml
```

{{% /hide %}}

#### Deleting a Setter

Setters may be deleted either manually (by editing the Kptfile directly), or
programmatically (through the `delete-setter` command). The `delete-setter`
command will:

1. delete an OpenAPI definition for a setter in the Kptfile
2. remove references to the setter definition on the resource fields

```yaml
# Kptfile -- original
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      x-k8s-cli:
        setter:
          name: "replicas"
          value: "3"
```

```yaml
# deployment.yaml -- original
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
```

<!-- @deleteSetter @verifyGuides-->

```sh
# delete a setter named "replicas"
kpt cfg delete-setter . replicas
```

```yaml
# Kptfile -- updated
openAPI:
```

```yaml
# deployment.yaml -- updated
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 3
```

#### OpenAPI Validations

Users can input any additional validation constraints during `create-setter`
operation in the form of openAPI schema. Relevant openAPI specification
constraints can be provided in json file format. The `set` operation validates
the input value against provided schema during setter creation and throws an
error if the input value doesn't meet any of the constraints. This example walks
you through the steps to work with openAPI validations.

<!-- @createSetterForValidation @verifyGuides-->

```sh
cat <<EOF >schema.json
{"maximum": 10, "type": "integer"}
EOF

# create setter with openAPI property constraints
kpt cfg create-setter . replicas 5 --schema-path schema.json
```

The command creates setter with the following definition

```yaml
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      maximum: 10
      type: integer
      x-k8s-cli:
        setter:
          name: replicas
          value: "5"
```

```sh
# try to set value not adhering to the constraints
kpt cfg set . replicas 11
```

```sh
error: The input value doesn't validate against provided OpenAPI schema:
validation failure list: replicas in body should be less than or equal to 10
```

{{% hide %}}

<!-- @testSetSetter @verifyGuides-->

```
kpt cfg set . replicas 11 || echo "Worked" | grep "Worked"
kpt cfg delete-setter . replicas
```

{{% /hide %}}

```sh
Example schema for integer validation

{
  "maximum": 10,
  "type": "integer",
  "minimum": 3,
  "format": "int64",
  "multipleOf": 2
}

Example schema for string validation

{
  "maxLength": 10,
  "type": "string",
  "minLength": 3,
  "pattern": "^[A-Za-z]+$",
  "enum": [
    "nginx",
    "ubuntu"
  ]
}

Example schema for array validation

{
  "maxItems": 10,
  "type": "array",
  "minItems": 3,
  "uniqueItems": true,
  "items": {
    "type": "string",
    "maxLength": 4
  }
}

```

Relevant resources for more information: [OpenAPI types]

#### Required setters

Package publisher can mark a setter as required to convey the consumer that the
setter value must be set before triggering live apply/preview operation
on the package. This example walks you through the steps to work with required setters.

```sh
# create a setter named "replicas" and mark it as required
kpt cfg create-setter . replicas 3 --required
```

```yaml
# deployment-foo.yaml
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
```

```yaml
# Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      x-k8s-cli:
        setter:
          name: "replicas"
          value: "3"
          required: true
```

```sh
# if you live apply/preview without setting the value
kpt live apply .

error: setter replicas is required but not set, please set it and try again
```

```sh
# set the replicas value
kpt cfg set hello-world/ replicas 4

kpt live apply hello-world/
# Success
```

#### Setting Lists

It is possible to create setters for fields which are a list of strings/integers.
The setter type must be `array`, and the reference must be on the list field.
The list setter will take variable args for its value rather than a single value.

**Note:** You should skip passing the value arg while creating array setters. `field`
flag is required for array setters.

<!-- @createResourceForListSetter @verifyGuides-->

```sh
cat <<EOF >example.yaml
# example.yaml
apiVersion: example.com/v1beta1
kind: Example
spec:
  list:
  - "a"
  - "b"
EOF
```

```yaml
# Kptfile
kind: Kptfile
```

<!-- @createListSetter @verifyGuides-->

```
kpt cfg create-setter . list --type array --field spec.list
```

```yaml
# example.yaml
apiVersion: example.com/v1beta1
kind: Example
spec:
  list: # {"$kpt-set":"list"}
    - "a"
    - "b"
```

```yaml
# Kptfile
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.list:
      type: array
      x-k8s-cli:
        setter:
          name: list
          listValues:
            - a
            - b
```

{{% hide %}}

<!-- @validateCreateListSetter @verifyGuides-->

```
grep "type: array" Kptfile
grep "listValues:" Kptfile
grep 'list: # {"$kpt-set":"list"}' example.yaml
```

{{% /hide %}}

<!-- @setListSetter @verifyGuides-->

```
kpt cfg set . list c d e
```

```yaml
# example.yaml
apiVersion: example.com/v1beta1
kind: Example
spec:
  list: # {"$kpt-set":"list"}
    - "c"
    - "d"
    - "e"
```

```yaml
# Kptfile
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.list:
      type: array
      x-k8s-cli:
        setter:
          name: list
          listValues:
            - "c"
            - "d"
            - "e"
```

{{% hide %}}

<!-- @validateSetListSetter @verifyGuides-->

```
grep '\- "d"' Kptfile
grep '\- "d"' example.yaml
```

{{% /hide %}}

[openapi types]: https://swagger.io/docs/specification/data-models/data-types/
[setters]: /concepts/setters/#setters
