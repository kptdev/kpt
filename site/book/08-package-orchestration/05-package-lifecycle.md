When a new package revision is created, it is in a **`Draft`** lifecycle stage,
where the package can be authored, including updating its contents.

Before a package can be deployed or cloned, it must be **`Published`**.
The approval flow is the process by which the package is advanced from
**`Draft`** to **`Proposed`** and finally **`Published`** lifecycle stage.

In the [previous section](./04-package-authoring) we created several packages,
let's explore how to publish some of them.

```sh
# List package revisions (the output was abbreviated to only include Draft)
# packages
$ kpt alpha rpkg get
NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b   istions       v1         false    Draft       deployments
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486   my-bucket     v1         false    Draft       deployments
deployments-93bb9ac8c2fb7a5759547a38f5f48b369f42d08a   new-package   v2         false    Draft       deployments
...
```

Now, in the role of the package author, we will propose two of those packages
to be published: `istions/v1` and `my-bucket/v2`.

```sh
# Propose two package revisions to be be published
$ kpt alpha rpkg propose \
  deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b \
  deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 \
  -ndefault

deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b proposed
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 proposed
```

?> Refer to the [propose command reference][rpkg-propose] for usage.

The two package revisions are now **`Proposed`**:

```sh
# Confirm the package revisions are now Proposed (the output was abbreviated
# to only show relevant packages)
$ kpt alpha rpkg get      
NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b   istions       v1         false    Proposed    deployments
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486   my-bucket     v1         false    Proposed    deployments
deployments-93bb9ac8c2fb7a5759547a38f5f48b369f42d08a   new-package   v2         false    Draft       deployments
...
```

At this point, a person in the _platform administrator_ role, or even an
automated process, will review and either approve or reject the proposals.
To aid with the decision, the platform administrator may inspect the package
contents using the commands above, such as `kpt alpha rpkg pull`.

```sh
# Approve a proposal to publish istions/v1
$ kpt alpha rpkg approve deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b -ndefault
deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b approved

# Reject a proposal to publish a my-bucket/v1
$ kpt alpha rpkg reject deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 -ndefault
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 rejected
```

?> Refer to the [approve][rpkg-approve] and [reject][rpkg-reject] command
reference for usage.

> Approving a package revisions requires that the current user has been granted
> update access to the `approve` subresource of `packagerevisions`. This allows
> for giving only a limited set of users permission to approve package revisions.

Now, confirm lifecycle stages of the package revisions:

```sh
# Confirm package revision lifecycle stages after approvals (output was
# abbreviated to display only relevant package revisions):
$ kpt alpha rpkg get
NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-98bc9a49246a5bd0f4c7a82f3d07d0d2d1293cd0   istions       main       false    Published   deployments
deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b   istions       v1         true     Published   deployments
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486   my-bucket     v1         false    Draft       deployments
deployments-93bb9ac8c2fb7a5759547a38f5f48b369f42d08a   new-package   v2         false    Draft       deployments
...
```

The rejected proposal returned the package to **`Draft`**, and the approved
proposal resulted in **`Published`** package revision.

You may have noticed that a `main` revision of the istions package appeared.
When a package is approved, Porch will commit it into the branch which was
provided at the repository registration (`main` in this case) and apply a tag.
As a result, the package revision exists in two locations - tag, and the `main`
branch.

[rpkg-propose]: /reference/cli/alpha/rpkg/propose/
[rpkg-approve]: /reference/cli/alpha/rpkg/approve/
[rpkg-reject]: /reference/cli/alpha/rpkg/reject/
