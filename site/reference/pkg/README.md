---
title: "Pkg"
linkTitle: "pkg"
weight: 1
type: docs
description: >
   Fetch, update, and sync configuration files using git
---
<!--mdtogo:Short
    Fetch, update, and sync configuration files using git
-->

<!--mdtogo:Long-->
The `pkg` command group contains subcommands for fetching and updating kpt package
from git repositories. They are focused on providing porcelain on top of 
workflows which would otherwise require wrapping git to pull clone subdirectories
and perform updates by merging resources rather than files.
<!--mdtogo-->
