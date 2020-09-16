---
title: "Set Dependency"
linkTitle: "Set Dependency"
type: docs
draft: true
description: >
  Set a subpackage as a dependency to a Kptfile
---

<!--mdtogo:Short
   Set a subpackage as a dependency to a Kptfile
-->

`kpt pkg set-dependency` can be used to declare a [subpackage] as `dependency` to Kptfile
of a parent package. `dependencies` section in Kptfile holds the metadata related to
`subpackages`

{{% pageinfo color="primary" %}}
Note: command must be run from within the directory containing Kptfile
to be updated.
{{% /pageinfo %}}

### Examples

<!--mdtogo:Examples-->

#### Create a new package and add a dependency to it

```sh
# init a package
kpt pkg init .

# add a dependency to the package
kpt pkg set-dependency https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set \
    hello-world
```

<!--mdtogo-->

### Synopsis

<!--mdtogo:Long-->

```
kpt pkg set-dependency REPO_URI[.git]/PKG_PATH[@VERSION] LOCAL_DEST_DIRECTORY [flags]

REPO_URI:
  URI of a git repository containing 1 or more packages as subdirectories.
  In most cases the .git suffix should be specified to delimit the REPO_URI
  from the PKG_PATH, but this is not required for widely recognized repo
  prefixes.  If get cannot parse the repo for the directory and version,
  then it will print an error asking for '.git' to be specified as part of
  the argument.
  e.g. https://github.com/kubernetes/examples.git
  Specify - to read Resources from stdin and write to a LOCAL_DEST_DIRECTORY

PKG_PATH:
  Path to remote subdirectory containing Kubernetes Resource configuration
  files or directories.  Defaults to the root directory.
  Uses '/' as the path separator (regardless of OS).
  e.g. staging/cockroachdb

VERSION:
  A git tag, branch, ref or commit for the remote version of the package to
  fetch.  Defaults to the repository master branch.
  e.g. @master

LOCAL_DEST_DIRECTORY:
  The local directory to write the package to. e.g. ./my-cockroachdb-copy

    * If the directory does NOT exist: create the specified directory and write
      the package contents when get/update command is invoked
    * If the directory DOES exist and contains a Kptfile: update the subpackage
      when update command is invoked
    * If the directory DOES exist and does not contain a Kptfile: Throw an error
```

#### Flags

```
--strategy:
  Controls how changes to the local package are handled.
  Defaults to resource-merge.

    * resource-merge: perform a structural comparison of the original /
      updated Resources, and merge the changes into the local package.
      See `kpt help apis merge3` for details on merge.
    * fast-forward: fail without updating if the local package was modified
      since it was fetched.
    * alpha-git-patch: use 'git format-patch' and 'git am' to apply a
      patch of the changes between the source version and destination
      version.
      REQUIRES THE LOCAL PACKAGE TO HAVE BEEN COMMITTED TO A LOCAL GIT REPO.
    * force-delete-replace: THIS WILL WIPE ALL LOCAL CHANGES TO
      THE PACKAGE.  DELETE the local package at local_pkg_dir/ and replace
      it with the remote version.
```

<!--mdtogo-->

[subpackage]: https://googlecontainertools.github.io/kpt/concepts/packaging/#subpackages
