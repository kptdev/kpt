# Annotations Reference

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

[config.kubernetes.io/depends-on]: /reference/annotations/depends-on/
[config.kubernetes.io/apply-time-mutation]: /reference/annotations/apply-time-mutation/
[config.kubernetes.io/local-config]: /reference/annotations/local-config/
