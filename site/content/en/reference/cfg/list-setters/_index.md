---
title: "List Setters"
linkTitle: "list-setters"
weight: 4
type: docs
description: >
   List setters for a package
---
<!--mdtogo:Short
    List setters for a package
-->

{{< asciinema key="cfg-set" rows="10" preload="1" >}}

List setters displays the setters that may be provided to the set command.
It also displays:

- The current setter value
- A record of who last set the value
- A description of the value or setter
- The name of fields that would be updated by calling set

See [create-setter] and [create-subst] for how setters and substitutions
are defined in a Kptfile.

### Examples
<!--mdtogo:Examples-->
```sh
# list the setters in the hello-world package
kpt cfg list-setters hello-world/

  NAME     VALUE    SET BY    DESCRIPTION   COUNT  
replicas   4       isabella   good value    1
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```sh
kpt cfg list-setters DIR [NAME]

DIR
  Path to a package directory

NAME
  Optional.  The name of the setter to display.
```
<!--mdtogo-->

[create-setter]: ../create-setter/
[create-subst]: ../create-subst/
