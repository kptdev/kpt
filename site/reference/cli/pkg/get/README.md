---
title: "`get`"
linkTitle: "get"
type: docs
description: >
  Fetch a package from a git repo.
---

<!--mdtogo:Short
    Fetch a package from a git repo.
-->

`get` fetches a remote package from a git subdirectory and writes it to a new
local directory.

### Synopsis

<!--mdtogo:Long-->

```
kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] [LOCAL_DEST_DIRECTORY] [flags]
```

#### Args

```
REPO_URI:
  URI of a git repository containing 1 or more packages as subdirectories.
  In most cases the .git suffix should be specified to delimit the REPO_URI
  from the PKG_PATH, but this is not required for widely recognized repo
  prefixes. If get cannot parse the repo for the directory and version,
  then it will print an error asking for '.git' to be specified as part of
  the argument.

PKG_PATH:
  Path to remote subdirectory containing Kubernetes resource configuration
  files or directories. Defaults to the root directory.
  Uses '/' as the path separator (regardless of OS).
  e.g. staging/cockroachdb

VERSION:
  A git tag, branch, ref or commit for the remote version of the package
  to fetch. Defaults to the default branch of the repository.

LOCAL_DEST_DIRECTORY:
  The local directory to write the package to. Defaults to a subdirectory of the
  current working directory named after the upstream package.
```

#### Flags

```
--strategy:
  Defines which strategy should be used to update the package. It defaults to
  'resource-merge'.

    * resource-merge: Perform a structural comparison of the original /
      updated resources, and merge the changes into the local package.
    * fast-forward: Fail without updating if the local package was modified
      since it was fetched.
    * force-delete-replace: Wipe all the local changes to the package and replace
      it with the remote version.

--for-deployment:
  (Experimental) indicates if the fetched package is a deployable instance that
  will be deployed to a cluster.
  It is `false` by default.
```

#### Env Vars

```
KPT_CACHE_DIR:
  Controls where to cache remote packages when fetching them.
  Defaults to <HOME>/.kpt/repos/
  On macOS and Linux <HOME> is determined by the $HOME env variable, while on
  Windows it is given by the %USERPROFILE% env variable.
```

<!--mdtogo-->

### Examples

{{% hide %}}

<!-- @makeWorkplace @verifyExamples-->

```
# Set up workspace for the test.
TEST_HOME=$(mktemp -d)
cd $TEST_HOME
```

{{% /hide %}}

<!--mdtogo:Examples-->

<!-- @pkgGet @verifyExamples-->

```shell
# Fetch package cockroachdb from github.com/kubernetes/examples/staging/cockroachdb
# This creates a new subdirectory 'cockroachdb' for the downloaded package.
$ kpt pkg get https://github.com/kubernetes/examples.git/staging/cockroachdb@master
```

<!-- @pkgGet @verifyExamples-->

```shell
# Fetch package cockroachdb from github.com/kubernetes/examples/staging/cockroachdb
# This will create a new directory 'my-package' for the downloaded package if it
# doesn't already exist.
$ kpt pkg get https://github.com/kubernetes/examples.git/staging/cockroachdb@master ./my-package/
```

<!-- @pkgGet @verifyExamples-->

```shell
# Fetch package examples from github.com/kubernetes/examples at the specified
# git hash.
# This will create a new directory 'examples' for the package.
$ kpt pkg get https://github.com/kubernetes/examples.git/@6fe2792
```

<!-- @pkgGet @verifyExamples-->

```shell
# Create a deployable instance of examples package from github.com/kubernetes/examples
# This will create a new directory 'examples' for the package.
$ kpt pkg get https://github.com/kubernetes/examples.git/@6fe2792 --for-deployment
```

<!--mdtogo-->
