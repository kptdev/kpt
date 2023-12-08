There are several ways to author a package revision, including creating
a completely new one, cloning an existing package, or creating a new revision
of an existing package. In this section we will explore the different ways to
author package revisions, and explore how to modify package contents.

## Create a new package revision

Create a new package revision in a repository managed by Porch:

```sh
# Initialize a new (empty) package revision:
$ kpt alpha rpkg init new-package --repository=deployments --revision=v1 -ndefault

deployments-c32b851b591b860efda29ba0e006725c8c1f7764 created

# List the available package revisions.
$ kpt alpha rpkg get

NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-c32b851b591b860efda29ba0e006725c8c1f7764   new-package   v1         false    Draft       deployments
kpt-samples-da07e9611f9b99028f761c07a79e3c746d6fc43b   basens        main       false    Published   kpt-samples
kpt-samples-afcf4d1fac605a60ba1ea4b87b5b5b82e222cb69   basens        v0         true     Published   kpt-samples
...
```

?> Refer to the [init command reference][rpkg-init] for usage.

You can see the `new-package` is created in the `Draft` lifecycle stage. This
means that the package is being authored.

> You may notice that the name of the package revision
> `deployments-c32b851b591b860efda29ba0e006725c8c1f7764` was assigned
> automatically. Packages in a git repository may be located in subdirectories
> and to make sure Porch works well with the rest of the Kubernetes ecosystem,
> the resource names must meet Kubernetes requirements. The resource names
> assigned by Porch are stable, and computed as hash of the repository name,
> directory path within the repository, and revision.

The contents of the new package revision are the same as if it was created using
the [`kpt pkg init`](/book/03-packages/06-creating-a-package) command, except it
was created by the Package Orchestration service in your repository.

In fact, if you check your Git repository, you will see a new branch called
`drafts/new-package/v1` which Porch created for the draft authoring. You will
also see one or more commits made into the branch by Porch on your behalf.

## Clone an existing package

Another way to create a new package revision is by cloning an already existing
package. The existing package is referred to as *upstream* and the newly created
package is *downstream*.

Use `kpt alpha rpkg clone` command to create a new *downstream* package
`istions` by cloning the sample `basens/v0` package revision:

```sh
# Clone an upstream package to create a downstream package
$ kpt alpha rpkg clone \
  kpt-samples-afcf4d1fac605a60ba1ea4b87b5b5b82e222cb69 \
  istions \
  --repository=deployments -ndefault

deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b created

# Confirm the package revision was created
kpt alpha rpkg get deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b -ndefault
NAME                                                   PACKAGE   REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b   istions   v1         false    Draft       deployments
```

?> Refer to the [clone command reference][rpkg-clone] for usage.

Cloning a package using the Package Orchestration service is an action similar to
[`kpt pkg get`](/book/03-packages/01-getting-a-package) command. Porch will
create the appropriate upstream package links in the new package's `Kptfile`.
Let's take a look:

```sh
# Examine the new package's upstream link (the output has been abbreviated):
$ kpt alpha rpkg pull deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b -ndefault

kpt alpha rpkg pull deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b -ndefault
apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: istions
  upstream:
    type: git
    git:
      repo: https://github.com/GoogleContainerTools/kpt-samples.git
      directory: basens
      ref: basens/v0
  upstreamLock:
    type: git
    git:
      repo: https://github.com/GoogleContainerTools/kpt-samples.git
      directory: basens
      ref: basens/v0
      commit: 026dfe8e3ef8d99993bc8f7c0c6ba639faa9a634
  info:
    description: kpt package for provisioning namespace
...
```

You can find out more about the `upstream` and `upstreamLock` sections of the
`Kptfile` in an [earlier chapter](/book/03-packages/01-getting-a-package)
of the book.

> A cloned package must be created in a repository in the same namespace as
> the source package. Cloning a package with the Package Orchestration Service
> retains a reference to the upstream package revision in the clone, and
> cross-namespace references are not allowed. Package revisions in repositories
> in other namespaces can be cloned using a reference directly to the underlying
> oci or git repository as described below.

You can also clone a package from a repository that is _not_ registered with
Porch, for example:

```sh
# Clone a package from Git repository directly (repository is not registered)
$ kpt alpha rpkg clone \
  https://github.com/GoogleCloudPlatform/blueprints.git/catalog/bucket@main my-bucket \
  --repository=deployments \
  --namespace=default

deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 created

# Confirm the package revision was created
$ kpt alpha rpkg get deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486 -ndefault
NAME                                                   PACKAGE     REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-8baf4892d6bdeda0f26ef4b1088fddb85c5a2486   my-bucket   v1         false    Draft       deployments
```

## Create a new revision of an existing package

Finally, with Porch you can create a new revision of an existing,
**`Published`** package. All the package revisions in your repository are
**`Draft`** revisions and need to be published first. We will cover the package
approval flow in more detail in the next section. For now we will quickly
propose and approve one of our draft package revisions and create a new revision
from it.

```sh
# Propose the package draft to be published
$ kpt alpha rpkg propose deployments-c32b851b591b860efda29ba0e006725c8c1f7764 -ndefault
deployments-c32b851b591b860efda29ba0e006725c8c1f7764 proposed

# Approve the proposed package revision for publishing
$ kpt alpha rpkg approve deployments-c32b851b591b860efda29ba0e006725c8c1f7764 -ndefault
deployments-c32b851b591b860efda29ba0e006725c8c1f7764 approved
```

You now have a **`Published`** package revision in the repository managed by Porch
and next you will create a new revision of it. A **`Published`** package is ready
to be used, such as deployed or copied.

```sh
# Confirm the package is published:
$ kpt alpha rpkg get deployments-c32b851b591b860efda29ba0e006725c8c1f7764 -ndefault
NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-c32b851b591b860efda29ba0e006725c8c1f7764   new-package   v1         true     Published   deployments
```

Copy the existing, **`Published`** package revision to create a **`Draft`** of
a new package revision that you can further customize:

```sh
# Copy the published package:
$ kpt alpha rpkg copy deployments-c32b851b591b860efda29ba0e006725c8c1f7764 \
  -ndefault --revision v2
deployments-93bb9ac8c2fb7a5759547a38f5f48b369f42d08a created

# List all revisions of the new-package that we just copied:
$ kpt alpha rpkg get --name new-package
NAME                                                   PACKAGE       REVISION   LATEST   LIFECYCLE   REPOSITORY
deployments-af86ae3c767b0602a198856af513733e4e37bf10   new-package   main       false    Published   deployments
deployments-c32b851b591b860efda29ba0e006725c8c1f7764   new-package   v1         true     Published   deployments
deployments-93bb9ac8c2fb7a5759547a38f5f48b369f42d08a   new-package   v2         false    Draft       deployments
```

?> Refer to the [copy command reference][rpkg-copy] for usage.

Unlike `clone` of a package which establishes the upstream-downstream
relationship between the respective packages, and updates the `Kptfile`
to reflect the relationship, the `copy` command does *not* change the
upstream-downstream relationships. The copy of a package shares the same
upstream package as the package from which it was copied. Specifically,
in this case both `new-package/v1` and `new-package/v2` have identical contents,
including upstream information, and differ in revision only.

## Editing package revision resources

One of the driving motivations for the Package Orchestration service is enabling
WYSIWYG authoring of packages, including their contents, in highly usable UIs.
Porch therefore supports reading and updating package *contents*.

In addition to using a [UI](/guides/namespace-provisioning-ui) with Porch, we
can change the package contents by pulling the package from Porch onto the local
disk, make any desired changes, and then pushing the updated contents to Porch.

```sh
# Pull the package contents of istions/v1 onto the local disk:
$ kpt alpha rpkg pull deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b ./istions -ndefault
```

?> Refer to the [pull command reference][rpkg-pull] for usage.

The command downloaded the `istions/v1` package revision contents and saved
them in the `./istions` directory. Now you will make some changes.

First, note that even though Porch updated the namespace name (in
`namespace.yaml`) to `istions` when the package was cloned, the `README.md`
was not updated. Let's fix it first.

Open the `README.md` in your favorite editor and update its contents, for
example:

```
# istions

## Description
kpt package for provisioning Istio namespace
```

In the second change, add a new mutator to the `Kptfile` pipeline. Use the
[set-labels](https://catalog.kpt.dev/set-labels/v0.1/) function which will add
labels to all resources in the package. Add the following mutator to the
`Kptfile` `pipeline` section:

```yaml
  - image: gcr.io/kpt-fn/set-labels:v0.1.5
    configMap:
      color: orange
      fruit: apple
```

The whole `pipeline` section now looks like this:

```yaml
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/set-namespace:v0.4.1
    configPath: package-context.yaml
  - image: gcr.io/kpt-fn/apply-replacements:v0.1.1
    configPath: update-rolebinding.yaml
  - image: gcr.io/kpt-fn/set-labels:v0.1.5
    configMap:
      color: orange
      fruit: apple
```

Save the changes and push the package contents back to the server:

```sh
# Push updated package contents to the server
$ kpt alpha rpkg push deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b ./istions -ndefault
```

?> Refer to the [push command reference][rpkg-push] for usage.

Now, pull the contents of the package revision again, and inspect one of the
configuration files.

```sh
# Pull the updated package contents to local drive for inspection:
$ kpt alpha rpkg pull deployments-eeb52a8072ca2602e7ee27f3c56ad6344b024f5b ./updated-istions -ndefault

# Inspect updated-istions/namespace.yaml
$ cat updated-istions/namespace.yaml 

apiVersion: v1
kind: Namespace
metadata:
  name: istions
  labels:
    color: orange
    fruit: apple
spec: {}
```

The updated namespace now has new labels! What happened?

Whenever package is updated during the authoring process, Porch automatically
re-renders the package to make sure that all mutators and validators are
executed. So when we added the new `set-labels` mutator, as soon as we pushed
the updated package contents to Porch, Porch re-rendered the package and
the `set-labels` function applied the labels we requested (`color: orange` and
`fruit: apple`).

## Summary of package authoring

In this section we reviewed how to use Porch to author packages, including

* creating a new package ([`kpt alpha rpkg init`][rpkg-init])
* cloning an existing package ([`kpt alpha rpkg clone`][rpkg-clone])
* creating a new revision of an existing package
  ([`kpt alpha rpkg copy`][rpkg-copy])
* pulling package contents for local editing
  ([`kpt alpha rpkg pull`][rpkg-pull])
* and pushing updated package contents to Porch
  ([`kpt alpha rpkg push`][rpkg-push])

[rpkg-init]: /reference/cli/alpha/rpkg/init/
[rpkg-get]: /reference/cli/alpha/rpkg/get/
[rpkg-clone]: /reference/cli/alpha/rpkg/clone/
[rpkg-copy]: /reference/cli/alpha/rpkg/copy/
[rpkg-pull]: /reference/cli/alpha/rpkg/pull/
[rpkg-push]: /reference/cli/alpha/rpkg/push/
