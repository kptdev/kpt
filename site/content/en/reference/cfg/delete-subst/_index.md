---
title: "Delete-subst"
linkTitle: "delete-subst"
weight: 4
type: docs
description: >
   Delete a substitution
---
<!--mdtogo:Short
    Delete a substitution
-->

Substitutions provide a solution for template-free substitution of field values
built on top of [setters]. They enable substituting values into part of a
field, including combining multiple setters into a single value.

See the [creating substitutions] guide for more info on creating
substitutions.

The created substitutions can be deleted using `delete-subst` command.

### Examples
<!--mdtogo:Examples-->
```sh
# delete a substitution image-tag
kpt cfg delete-subst DIR/ image-tag
```

<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```sh
kpt cfg delete-subst DIR NAME

DIR:
  Path to a package directory

NAME:
  The name of the substitution to delete. e.g. image-tag

```

<!--mdtogo-->

```sh

--recurse-subpackages, -R
  Delete substitution recursively in all the nested subpackages

```

[setters]: ../create-setter/
[creating substitutions]: ../../../guides/producer/substitutions/
