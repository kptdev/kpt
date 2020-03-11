---
title: "Create-subst"
linkTitle: "create-subst"
weight: 4
type: docs
description: >
   Create a substitution for one or more fields
---

{{< asciinema key="cfg-create-subst" rows="10" preload="1" >}}

Substitutions provide a solution for template-free substitution of field values
built on top of [setters].  They enable substituting values into part of a
field, including combining multiple setters into a single value.

Much like setters, substitutions are defined using OpenAPI.

Substitutions may be invoked to programmatically modify the configuration
using `kpt cfg set` to substitute values which are derived from the setter.

Substitutions are computed by substituting setter values into a pattern.
They are composed of 2 parts: a pattern and a list of values.

- The pattern is a string containing markers which will be replaced with
  1 or more setter values.
- The values are pairs of markers and setter references.  The *set* command
  retrieves the values from the referenced setters, and replaces the markers
  with the setter values.

#### Creating a Substitution

Substitution may be created either manually (by editing the Kptfile directly),
or programmatically.  The `create-subst` command will:

1. create a new OpenAPI definition for a substition in the Kptfile
2. create references to the substitution definition on the resource fields

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
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9 # {"$ref":"#/definitions/io.k8s.cli.substitutions.image-value"}
```

```sh
# create an image substitution and a setter that populates it
kpt cfg create-subst hello-world/ image-tag nginx:1.7.9 \
  --pattern nginx:TAG_SETTER --value TAG_SETTER=tag
```

```yaml
# Kptfile -- updated
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.7.9"
    io.k8s.cli.substitutions.image-value:
      x-k8s-cli:
        substitution:
          name: image-value
          pattern: nginx:TAG_SETTER
          values:
          - marker: TAG_SETTER
            ref: '#/definitions/io.k8s.cli.setters.tag'
```

```yaml
# deployment.yaml -- updated
kind: Deployment
metadata:
  name: foo
spec:
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9 # {"$ref":"#/definitions/io.k8s.cli.substitutions.image-value"}
```

This substitution defines how the value for a field may be produced by
substituting the `tag` setter value into the pattern.

Any time the `tag` value is changed via the *set* command, then then
substitution value will be re-calculated for referencing fields.

#### Creation semantics

By default create-subst will create referenced setters if they do not already
exist.  It will infer the current setter value from the pattern and value.

If setters already exist before running the create-subst command, then those
setters are used and left unmodified.

If a setter does not exist and create-subst cannot infer the setter value,
then it will throw and error, and the setter must be manually created.

#### Invoking a Substitution

Substitutions are invoked by running `kpt cfg set` on a setter used by the
substitution.

```sh
kpt cfg set hello-world/ image 1.8.1
```

```yaml
# deployment.yaml -- updated
kind: Deployment
metadata:
  name: foo
spec:
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.8.1 # {"$ref":"#/definitions/io.k8s.cli.substitutions.image-value"}
```

**Note**: When setting a field through a substitution, the names of the setters
are used *not* the name of the substitution.  The name of the substitution is
*only used in the configuration field references*.

### Examples

```sh

# Automatically create setters when creating the substitution, inferring
# the setter values.
#
# 1. create a substitution derived from 2 setters.  The user will never
# call the substitution directly, instead it will be computed when the
# setters are used.
kpt cfg create-subst DIR/ image-tag nginx:v1.7.9 \
  --pattern IMAGE_SETTER:TAG_SETTER \
  --value IMAGE_SETTER=nginx \
  --value TAG_SETTER=v1.7.9

# 2. update the substitution value by setting one of the 2 setters it is
# computed from
kpt cfg set tag v1.8.0

# Manually create setters and substitution.  This is preferred to configure
# the setters with a type, description, set-by, etc.
#
# 1. create the setter for the image name -- set the field so it isn't
# referenced
kpt cfg create-setter DIR/ image nginx --field "none" \
    --set-by "package-default"

# 2. create the setter for the image tag -- set the field so it isn't
# referenced
kpt cfg create-setter DIR/ tag v1.7.9 --field "none" \
    --set-by "package-default"

# 3. create the substitution computed from the image and tag setters
kpt cfg create-subst DIR/ image-tag nginx:v1.7.9 \
  --pattern IMAGE_SETTER:TAG_SETTER \
  --value IMAGE_SETTER=nginx \
  --value TAG_SETTER=v1.7.9

# 4. update the substitution value by setting one of the setters
kpt cfg set tag v1.8.0
```

### Synopsis

    kpt cfg create-subst DIR NAME VALUE --pattern PATTERN --value MARKER=SETTER

    DIR
      Path to a package directory

    NAME
      The name of the substitution to create.  This is simply the unique key
      which is referenced by fields which have the substitution applied.
      e.g. image-substitution

    VALUE
      The current value of the field that will have PATTERN substituted.
      e.g. nginx:1.7.9

    PATTERN
      A string containing one or more MARKER substrings which will be
      substituted for setter values.  The pattern may contain multiple
      different MARKERS, the same MARKER multiple times, and non-MARKER
      substrings.
      e.g. IMAGE_SETTER:TAG_SETTER

[setters]: create-setter
