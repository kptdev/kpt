---
title: "Destroy"
linkTitle: "destroy"
type: docs
description: >
   Remove all previously applied resources in a package from the cluster
---

The destroy command removes all files belonging to a package from the cluster.

### Synopsis

    kpt live destroy DIR

#### Args

    DIR:
      Path to a package directory.  The directory must contain exactly
      one ConfigMap with the grouping object annotation.
