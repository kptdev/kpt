# Reference

- [CLI Reference](/reference/cli/)
- [Schema Reference](/reference/schema/)
<!--mdtogo:Short
    Overview of kpt commands
-->

<!--mdtogo:Long-->

All kpt commands follow this general synopsis:

```
kpt <group> <command> <positional args> [PKG_PATH] [flags]
```

kpt functionality is divided into three command groups:

| Group   | Description                                                             |
| --------| ------------------------------------------------------------------------|
| pkg     | get, update, and describe packages with resources.                      |
| fn      | generate, transform, validate packages using containerized functions.   |
| live    | deploy local configuration packages to a cluster.                       |



<!--mdtogo-->

{{% hide %}}

<!-- @makeWorkplace @verifyExamples-->
```
# Runs the script which contains all verify functions. 
source ./scripts/setupVerify.sh
```

{{% /hide %}}
