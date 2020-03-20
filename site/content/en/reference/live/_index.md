---
title: "Live"
linkTitle: "live"
weight: 3
type: docs
description: >
   Reconcile configuration files with the live state
---
<!--mdtogo:Short
    Reconcile configuration files with the live state
-->

{{< asciinema key="live" rows="10" preload="1" >}}

<!--mdtogo:Long-->
| Reads From              | Writes To                |
|-------------------------|--------------------------|
| local files             | cluster                  |
| cluster                 | stdout                   |

Live contains the next-generation versions of apply related commands for
deploying local configuration packages to a cluster.
<!--mdtogo-->
