---
title: "Command Reference"
linkTitle: "Command Reference"
type: docs
weight: 40
menu:
  main:
    weight: 3
description: >
  Overview of kpt commands
---

<!--mdtogo:Short
    Overview of kpt commands
-->

<!--mdtogo:Long-->

Usage: kpt \<group> \<command> \<positional args> [PKG_PATH] [flags]

kpt functionality is divided into the following command groups, each of
which operates on a particular set of entities, with a consistent command
syntax and pattern of inputs and outputs.

| Group   | Description                                                             |
| --------| ------------------------------------------------------------------------|
| pkg     | get, update, and describe packages with resources.                      |
| fn      | generate, transform, validate packages using containerized functions.   |
| live    | deploy local configuration packages to a cluster.                       |

<!--mdtogo-->
