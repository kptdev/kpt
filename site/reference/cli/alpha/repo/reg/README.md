---
title: "`reg`"
linkTitle: "reg"
type: docs
description: >
  Registers a package repository with Package Orchestrator.
---

<!--mdtogo:Short
    Registers a package repository with Package Orchestrator.
-->

`reg` registers a package repository (either Git or OCI) with the Package Orchestrator.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha repo reg[ister] REPOSITORY [flags]
```

#### Args

```
REPOSITORY:
  Address of the repository to register. Required argument.
```

#### Flags

```
--description
  Brief description of the package repository.

--name
  Name of the package repository. If unspecified, will use the name portion (last segment) of the repository URL.

--title
  Title of the package repository.

--repo-username
  Username for repository authentication.

--repo-password
  Password for repository authentication.

```

<!--mdtogo-->