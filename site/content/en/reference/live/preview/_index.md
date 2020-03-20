---
title: "Preview"
linkTitle: "preview"
type: docs
description: >
   Preview prints the changes apply would make to the cluster
---
<!--mdtogo:Short
    Preview prints the changes apply would make to the cluster
-->

The preview command will run through the same steps as apply, but 
it will only print what would happen when running apply against the current
live cluster state. 

### Synopsis
<!--mdtogo:Long-->
    kpt live preview DIRECTORY [flags]

#### Args

    DIRECTORY:
      One directory that contain k8s manifests. The directory
      must contain exactly one ConfigMap with the grouping object annotation.

#### Flags

    --destroy:
      If true, dry-run deletion of all resources.
<!--mdtogo-->
