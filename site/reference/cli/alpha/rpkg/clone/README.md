---
title: "`clone`"
linkTitle: "clone"
type: docs
description: >
  Create a clone of an existing package revision.
---

<!--mdtogo:Short
    Create a clone of an existing package revision.
-->

`clone` creates a new package revision by cloning an existing one. The
new package revision will keep a reference to the source that can be used
to pull in updates.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg clone SOURCE_PACKAGE_REV TARGET_PACKAGE_NAME [flags]
```

#### Args

```
SOURCE_PACKAGE_REV:
  The source package that will be cloned to create the new package revision.
  The types of sources are supported:

    * OCI: A URI to a OCI repository must be provided. 
      oci://oci-repository/package-name
    * Git: A URI to a git repository must be provided.
      https://git-repository.git/package-name
    * Package: The name of a package revision already available in the
      repository.
      blueprint-e982b2196b35a4f5e81e92f49a430fe463aa9f1a

TARGET_PACKAGE_NAME:
  The name of the new package.

```

#### Flags

```
--directory
  Directory within the repository where the upstream
  package revision is located. This only applies if the source package is in git
  or oci.

--ref
  Ref in the repository where the upstream package revision
  is located (branch, tag, SHA). This only applies when the source package
  is in git.

--repository
  Repository to which package revision will be cloned
  (downstream repository).

--workspace
  Workspace for the new package. The default value is v1.

--strategy
  Update strategy that should be used when updating the new
  package revision. Must be one of: resource-merge, fast-forward,  or 
  force-delete-replace. The default value is resource-merge.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# clone the blueprint-e982b2196b35a4f5e81e92f49a430fe463aa9f1a package and create a new package revision called
# foo in the blueprint repository with a custom workspaceName.
$ kpt alpha rpkg clone blueprint-e982b2196b35a4f5e81e92f49a430fe463aa9f1a foo --repository blueprint --workspace=first-draft
```

```shell
# clone the git repository at https://github.com/repo/blueprint.git at reference base/v0 and in directory base. The new
# package revision will be created in repository blueprint and namespace default.
$ kpt alpha rpkg clone https://github.com/repo/blueprint.git bar --repository=blueprint --ref=base/v0 --namespace=default --directory=base
```

<!--mdtogo-->