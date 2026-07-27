---
title: Annotations Reference
linkTitle: Annotations Reference
description:
toc_hide: true
menu:
  main:
    parent: "References"
    weight: 30
---

The following annotations are used to configure kpt features:

| Annotation | Description |
| ---------- | ----------- |
| [config.kubernetes.io/depends-on]          | specifies one or more resource dependencies |
| [config.kubernetes.io/apply-time-mutation] | specifies one or more substitutions to make at apply time using dependencies as input |
| [config.kubernetes.io/local-config]        | specifies a resource to be skipped when applying |

The following annotations are used by kpt internally:

| Annotation | Description |
| ---------- | ----------- |
| `internal.config.kubernetes.io/path`   | specifies the resource's file path when formatted as a ResourceList |
| `internal.config.kubernetes.io/index`  | specifies the index of the resource in a file when formatted as a ResourceList |
| `config.kubernetes.io/merge-source`    | specifies the source of the resource during a three way merge |

## Path annotation details

The `internal.config.kubernetes.io/path` annotation specifies the file path of a
resource relative to the package directory (the directory containing the Kptfile
whose pipeline invoked the function).

When kpt reads resources from disk, it sets this annotation on each resource.
When a function creates new resources, it should set this annotation to a path
relative to the same package directory. kpt handles adjusting paths to be
relative to the root package before writing to disk.

For `kpt fn eval DIR`, the path is relative to DIR.

[config.kubernetes.io/depends-on]: /reference/annotations/depends-on/
[config.kubernetes.io/apply-time-mutation]: /reference/annotations/apply-time-mutation/
[config.kubernetes.io/local-config]: /reference/annotations/local-config/
