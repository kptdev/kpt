---
title: "Update"
linkTitle: "update"
type: docs
description: >
   Apply upstream package updates
---

{{< asciinema key="pkg-update" rows="10" preload="1" >}}

Update pulls in upstream changes and merges them into a local package.
Changes may be applied using one of several strategies.

{{% pageinfo color="primary" %}}
All changes must be committed to git before running update
{{% /pageinfo %}}

### Examples

```sh
# update my-package-dir/
git add . && git commit -m 'some message'
kpt pkg update my-package-dir/
```

```sh
# update my-package-dir/ to match the v1.3 branch or tag
git add . && git commit -m 'some message'
kpt pkg update my-package-dir/@v1.3
```

```sh
# update applying a git patch
git add . && git commit -m "package updates"
kpt pkg  update my-package-dir/@master --strategy alpha-git-patch
```

### Synopsis

    kpt pkg update LOCAL_PKG_DIR[@VERSION] [flags]

#### Args

    LOCAL_PKG_DIR:
      Local package to update.  Directory must exist and contain a Kptfile
      to be updated.

    VERSION:
  	  A git tag, branch, ref or commit.  Specified after the local_package
  	  with @ -- pkg@version.
      Defaults the local package version that was last fetched.

	  Version types:
        * branch: update the local contents to the tip of the remote branch
        * tag: update the local contents to the remote tag
        * commit: update the local contents to the remote commit

#### Flags

    --strategy:
      Controls how changes to the local package are handled.  Defaults to fast-forward.

        * resource-merge: perform a structural comparison of the original /
          updated Resources, and merge the changes into the local package.
        * fast-forward: fail without updating if the local package was modified
          since it was fetched.
        * alpha-git-patch: use 'git format-patch' and 'git am' to apply a
          patch of the changes between the source version and destination
          version.
        * force-delete-replace: WIPE ALL LOCAL CHANGES TO THE PACKAGE.
          DELETE the local package at local_pkg_dir/ and replace it
          with the remote version.

    -r, --repo:
      Git repo url for updating contents.  Defaults to the repo the package
      was fetched from.

    --dry-run
      Print the 'alpha-git-patch' strategy patch rather than merging it.

#### Env Vars

    KPT_CACHE_DIR:
      Controls where to cache remote packages when fetching them to update
      local packages.
      Defaults to ~/.kpt/repos/

