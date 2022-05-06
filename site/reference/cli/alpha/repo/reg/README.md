---
title: "`reg`"
linkTitle: "reg"
type: docs
description: >
  Register a package repository.
---

<!--mdtogo:Short
    Register a package repository.
-->

`reg` registers a new repository.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha repo reg REPOSITORY [flags]
```

#### Args

```
REPOSITORY:
  The URI for the registry. It can be either a git repository
  or an oci repository. For the latter, the URI must have the
  'oci://' prefix.
```

#### Flags

```
--branch:
  Branch within the repository where finalized packages are
  commited. The default is to use the 'main' branch.

--deployment:
  Tags the repository as a deployment repository. Packages in
  a deployment repository are considered ready for deployment.

--description:
  Description of the repository.

--directory:
  Directory within the repository where packages are found. The
  default is the root of the repository.

--name:
  Name of the repository. By default the last segment of the
  repository URL will be used as the name.

--repo-basic-username:
  Username for authenticating to a repository with basic auth.

--repo-basic-password:
  Password for authenticating to a repository with basic auth.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# register a new git repository with the name generated from the URI.
$ kpt alpha repo register https://github.com/platkrm/demo-blueprints.git --namespace=default
```

```shell
# register a new deployment repository with name foo.
$ kpt alpha repo register https://github.com/platkrm/blueprints-deployment.git --name=foo --deployment --namespace=bar
```

<!--mdtogo-->