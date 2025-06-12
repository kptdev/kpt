# CLI Reference

<!--mdtogo:Short
    Overview of kpt commands
-->

<!--mdtogo:Long-->

All kpt commands follow this general synopsis:

```
kpt <group> <command> <positional args> [PKG_PATH] [flags]
```

kpt functionality is divided into following command groups:

| Group   | Description                                                           |
| ------- | --------------------------------------------------------------------- |
| [pkg]   | get, update, and describe packages with resources.                    |
| [fn]    | generate, transform, validate packages using containerized functions. |
| [live]  | deploy local configuration packages to a cluster.                     |
| [alpha] | commands currently in alpha and might change without notice.          |

<!--mdtogo-->

[pkg]: /reference/cli/pkg/
[fn]: /reference/cli/fn/
[live]: /reference/cli/live/
[alpha]: /reference/cli/alpha/