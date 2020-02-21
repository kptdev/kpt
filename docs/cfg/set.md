## set

Set one or more field values using setters

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="/kpt/gifs/cfg-set.cast" speed="1" theme="solarized-dark" cols="100" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

    kpt tutorial cfg set

### Synopsis

    kpt cfg set DIR NAME VALUE

  DIR

    A directory containing Resource configuration.
    e.g. hello-world/

  NAME

    The name of the setter
    e.g. replicas

  VALUE

    The new value to set on fields
    e.g. 3

#### Setters

The *set* command modifies configuration fields using setters defined as OpenAPI definitions
in a Kptfile.  Setters are referenced by fields using line commands on the fields.  Fields
referencing a setter will have their value modified to match the setter value when the *set*
command is called.

If multiple fields may reference the same setter, all of the field's values will be
changed when the *set* command is called for that setter.

The *set* command must be run on a directory containing a Kptfile with setter definitions.
The list of setters configured for a package may be found using `kpt cfg list-setters`.

    kpt cfg set hello-world/ replicas 3

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

#### Description

Setters may have a description of the current value.  This may be defined along with
the value by specifying the `--description` flag.

#### SetBy

Setters may record who set the current value.  This may be defined along with the
value by specifying the `--set-by` flag.

#### Substitutions

Substitutions define field values which may be composed of one or more setters substituted
into a string pattern.  e.g. setting only the tag portion of the `image` field.

Anytime set is called for a setter used by a substitution, it will also modify the fields
referencing that substitution.

See `kpt cfg create-subst` for more information on substitutions.

### Examples

    # set replicas to 3 using the 'replicas' setter
    kpt cfg set hello-world/ replicas 3

    # set the replicas to 5 and include a description of the value
    kpt cfg set hello-world/ replicas 5 --description "need at least 5 replicas"

    # set the replicas to 5 and record who set this value
    kpt cfg set hello-world/ replicas 5 --set-by "mia"

    # set the tag portion of the image field to '1.8.1' using the 'tag' setter
    # the tag setter is referenced as a value by a substitution in the Kptfile
    kpt cfg set hello-world/ tag 1.8.1

###

[tutorial-script]: ../gifs/cfg-set.sh
