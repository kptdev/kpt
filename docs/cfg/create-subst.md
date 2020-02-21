## create-subst

Create or modify a field substitution

### Synopsis

    kpt cfg create-subst DIR NAME VALUE --pattern PATTERN --value MARKER=SETTER

  DIR

    A directory containing Resource configuration.
    e.g. hello-world/

  NAME

    The name of the substitution to create.  This is simply the unique key which is referenced
    by fields which have the substitution applied.
    e.g. image-substitution

  VALUE

    The current value of the field that will have PATTERN substituted.
    e.g. nginx:1.7.9

  PATTERN

    A string containing one or more MARKER substrings which will be substituted
    for setter values.  The pattern may contain multiple different MARKERS,
    the same MARKER multiple times, and non-MARKER substrings.
    e.g. IMAGE_SETTER:TAG_SETTER

#### Field Substitutions

Field substitutions are OpenAPI definitions that define how fields may be modified programmatically
using the *set* command.  The OpenAPI definitions for substitutions are defined in a Kptfile
and referenced by fields which they set through an OpenAPI reference as a line comment
(e.g. # {"$ref":"#/definitions/..."}).

Substitutions may be manually created by editing the Kptfile, or programmatically created using the
`create-subst` command.  The `create-subst` command will 1) create a new OpenAPI definition
for a substitution in the Kptfile, and 2) identify all fields matching the provided value and create
an OpenAPI reference to the substitution for each.

Field substitutions are computed by substituting setter values into a pattern.  They are
composed of 2 parts: a pattern and a list of values.

- The pattern is a string containing markers which will be replaced with 1 or more setter values.
- The values are pairs of markers and setter references.  The *set* command retrieves the values
  from the referenced setters, and replaces the markers with the setter values.
 
**The referenced setters must exist before creating the substitution.**

    # create or update a setter named image
    kpt create-setter hello-world/ image nginx

    # create or update a setter named tag
    kpt create-setter hello-world/ tag 1.7.9

    # create or update a substitution which is derived from concatenating the
    # image and tag setters
    kpt create-subst hello-world/ image-tag nginx:1.7.9 \
      --pattern IMAGE_SETTER:TAG_SETTER \
      --value IMAGE_SETTER=image \
      --value TAG_SETTER=tag

Example setter and substitution definitions in a Kptfile:

```yaml
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.7.9"
    io.k8s.cli.substitutions.image-value:
      x-k8s-cli:
        substitution:
          name: image-value
          pattern: IMAGE_SETTER:TAG_SETTER
          values:
          - marker: IMAGE_SETTER
            ref: '#/definitions/io.k8s.cli.setters.image'
          - marker: TAG_SETTER
            ref: '#/definitions/io.k8s.cli.setters.tag'
```

This substitution defines how a field value may be produced from the setters `image` and `tag`
by replacing the pattern substring *IMAGE_SETTER* with the value of the `image` setter, and
replacing the pattern substring *TAG_SETTER* with the value of the `tag` setter.  Any time
either the `image` or `tag` values are changed via *set*, the substitution value will be
re-calculated for referencing fields.

Example substitution reference from a field in a configuration file:

```yaml
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

The `image` field has a OpenAPI reference to the `image-value` substitution definition.  When
the *set* command is called, for either the `image` or `tag` setter, the substitution will
be recalculated, and the `image` field updated with the new value.

**Note**: when setting a field through a substitution, the names of the setters are used
*not* the name of the substitution.  The name of the substitution is *only used in field
references*.

### Examples

    # 1. create the setter for the image name -- set the field so it isn't referenced
    kpt cfg create-setter DIR/ image nginx --field "none"

    # 2. create the setter for the image tag -- set the field so it isn't referenced
    kpt cfg create-setter DIR/ tag v1.7.9 --field "none"

    # 3. create the substitution computed from the image and tag setters
    kpt cfg create-subst DIR/ image-tag nginx:v1.7.9 \
      --pattern IMAGE_SETTER:TAG_SETTER \
      --value IMAGE_SETTER=nginx \
      --value TAG_SETTER=v1.7.9

    # 4. update the substitution value by setting one of the setters
    kpt cfg set tag v1.8.0

### 
