---
title: "`unreg`"
linkTitle: "unreg"
type: docs
description: >
  Unregister a repository.
---

<!--mdtogo:Short
    Unregister a repository.
-->

`unreg` unregisters a repository.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha repo unreg REPOSITORY_NAME [flags]
```

#### Args

```
REPOSITORY_NAME:
  The name of a repository.
```

#### Flags

```
--keep-auth-secret:
  Keep the Secret object with auth information referenced by the repository.
  By default, it will be deleted when the repository is unregistered.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# unregister a repository and keep the auth secret.
$ kpt alpha repo unreg registered-repository --namespace=default --keep-auth-secret
```

<!--mdtogo-->