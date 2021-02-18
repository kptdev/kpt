---
title: "Set"
linkTitle: "set"
weight: 4
type: docs
description: >
   Set one or more field values
---
<!--mdtogo:Short
    Set one or more field values
-->

{{< asciinema key="cfg-set" rows="10" preload="1" >}}

The *set* command modifies configuration by setting or substituting
a user provided value into resource fields.  Which fields are set or
have values substituted is configured by line comments on the configuration
fields.

- Calling set may change multiple fields at once.
- To see the list of setters for a package run [list-setters].
- The *set* command may only be run on a directory containing a Kptfile.  

See [create-setter] and [create-subst] for more on how setters and substitutions
are defined in a Kptfile.

Example setter referenced from a field in a configuration file:

```yaml
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 3  # {"$ref":"#/definitions/io.k8s.cli.setters.replicas"}
```

One could the replicas field to 4 by running

``` sh
kpt cfg set hello-world/ replicas 4
```

#### Description

Setters may have a description of the current value.  This may be defined
along with the value by specifying the `--description` flag.

#### SetBy

Setters may record who set the current value.  This may be defined by
specifying the `--set-by` flag.  If unspecified the current
value for set-by will be cleared from the setter.

#### Substitutions

Substitutions define field values which may be composed of one or more setters
substituted into a string pattern.  e.g. setting only the tag portion of the
`image` field.

When set is called, it may also update substitutions which are derived from
the setter.

### Examples
<!--mdtogo:Examples-->
```sh
# set replicas to 3 using the 'replicas' setter
kpt cfg set hello-world/ replicas 3
```

```sh
# set the replicas to 5 and include a description of the value
kpt cfg set hello-world/ replicas 5 --description "need at least 5 replicas"
```

```sh
# set the replicas to 5 and record who set this value
kpt cfg set hello-world/ replicas 5 --set-by "mia"
```

```sh
# set the tag portion of the image field to '1.8.1' using the 'tag' setter
# the tag setter is referenced as a value by a substitution in the Kptfile
kpt cfg set hello-world/ tag 1.8.1
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```sh
kpt cfg set DIR NAME VALUE
```

#### Args

```sh
DIR
  Path to a package directory. e.g. hello-world/

NAME
  The name of the setter. e.g. replicas

VALUE
  The new value to set on fields. e.g. 3
```

#### Flags

```sh
--description
  Optional description about the value.

--set-by
  Optional record of who set the value.  Clears the last set-by
  value if unset.

--values
  Optional flag, the values of the setter to be set to
  e.g. used to specify values that start with '-'
```
<!--mdtogo-->

[create-setter]: ../create-setter/
[create-subst]: ../create-subst/
[list-setters]: ../list-setters/
