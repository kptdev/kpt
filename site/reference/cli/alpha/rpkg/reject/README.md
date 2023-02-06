---
title: "`reject`"
linkTitle: "reject"
type: docs
description: >
  Reject a proposal to publish or delete a package revision.
---

<!--mdtogo:Short
    Reject a proposal to publish or delete a package revision.
-->

`reject` closes a proposal for publishing or deleting a package revision.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg reject [PACKAGE_REV_NAME...] [flags]
```

#### Args

```
PACKAGE_REV_NAME...:
  The name of one or more package revisions. If more than
  one is provided, they must be space-separated.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# reject the proposal for package revision blueprint-8f9a0c7bf29eb2cbac9476319cd1ad2e897be4f9
$ kpt alpha rpkg reject blueprint-8f9a0c7bf29eb2cbac9476319cd1ad2e897be4f9 --namespace=default
```

<!--mdtogo-->