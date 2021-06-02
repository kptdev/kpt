An independent package records the exact commit where the local fork and the
upstream package diverged. This enables kpt to fetch any update to the upstream
package and merge it with local changes.

## Commit your local changes

Before you update the package, you want to commit your local changes.

First, to see the changes you've made to the fork of the upstream package:

```shell
$ git diff
```

If you're happy with the changes, commit them:

```shell
$ git add .; git commit -m "My changes"
```

## Update the package

For example, you can update to version `v0.4` of the `wordpress` package:

```shell
$ kpt pkg update wordpress@v0.4
```

This is a porcelain for manually updating the `upstream` section in the
`Kptfile` :

```yaml
upstream:
  type: git
  git:
    repo: https://github.com/GoogleContainerTools/kpt
    directory: /package-examples/wordpress
    # Change this from v0.3 to v0.4
    ref: v0.4
  updateStrategy: resource-merge
```

and then running:

```shell
$ kpt pkg update wordpress
```

The `update` command updates the local `wordpress` package and the dependent
`mysql` package to the upstream version `v0.4` by doing a 3-way merge between:

1. Original upstream commit
2. New upstream commit
3. Local (edited) package

<<<<<<< HEAD
Several different strategies are available to handle the merge. By default, the
`resource-merge` strategy is used which performs a structural comparison of the
resource using OpenAPI schema.
=======
Several different strategies are available to handle the merge. The default is the `resource-merge`
strategy.
>>>>>>> acf37cfc (Add details about the merge strategies to the Kpt book)

?> Refer to the [update command reference][update-doc] for usage.

## Commit the updated resources

Once you have successfully updated the package, commit the changes:

```shell
$ git add .; git commit -m "Updated wordpress to v0.4"
```

## Merge strategies

### Resource-merge

The `resource-merge` strategy performs a structural comparison of each resource using the
OpenAPI schema. So rather than performing a text-based merge, `kpt` leverages the
common structure of KRM resources.

#### Resource identity
In order to perform a per-resource merge, `kpt` needs to be able to match a resource in
the local package with the same resource in the upstream version of the package. It does
this matching based on the identity of a resource, which is the combination of group,
kind, name and namespace. So in our `wordpress` example, the identity of the`Deployment`
resource is:
```
group: apps
kind: Deployment
name: wordpress
namespace: ""
```
Changing the name and/or namespace of a resource is a pretty common way to customize
a package. In order to make sure this doesn't create problems during merge, `kpt` will
automatically adding the `# kpt-merge: <namespace>/<name>` comment on the `metadata` 
field of every resource when getting or updating a package. An example is the `Deployment`
resource from the `wordpress` package:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /wordpress
  name: wordpress
  labels:
    app: wordpress
...
```

#### Merge rules
`kpt` performs a 3-way merge for every resource. This means it will use the resource
in the local package, the updated resource from upstream, as well as the resource
at the version where the local and upstream package diverged (i.e.
common ancestor). When discussing the merge rules in detail, we will be referring to
the three different sources as local, upstream and origin.

In the discussion, we will be referring to non-associative and associative lists. A
non-associative list either has elements that are scalars or another list, or it has elements
that are mappings but without an associative key. An associative list has elements that are mappings and
one or more of the fields in the mappings are designated as associative keys. An associative key
(also sometimes referred to as a merge key) is used to identify the "same" elements in two
different lists for the purpose of merging them. `kpt` will primarily look for information about
any associative keys from the OpenAPI schema, but some fields are also automatically recognized as
associative keys:
* `mountPath`
* `devicePath`
* `ip`
* `type`
* `topologyKey`
* `name`
* `containerPort`

On the resource level, the merge rules are as follows:
* A resource present in origin and deleted from upstream will be deleted from local.
* A resource missing from origin and added in upstream will be added to local.
* A resource only in local will be kept without changes.
* A resource in both upstream and local will be merged into local.

On the field level, the rules differ based on the type of field:
* Scalars and non-associative lists:
    * If the field is present in either upstream or local and the value is `null`, remove the field from local.
    * If the field is unchanged between upstream and local, leave the local value unchanged.
    * If the field has been changed in both upstream and local, update local with the value from upstream.

* Mappings:
    * If the field is present in either upstream or local and the value is `null`, remove the field from local.
    * If the field is present only in local, leave the local value unchanged.
    * If the field is not present in local, add the delta between origin and upstream as the value in local.
    * If the field is present in both upstream and local, recursively merge the values between local, upstream and origin.

* Associative lists:
    * If the field is present in either upstream or local and the value is `null`, remove the field from local.
    * If the field is present only in local, leave the local value unchanged.
    * If the field is not present in local, add the delta between origin and upstream as the value in local.
    * If the field is present in both upstream and local, recursively merge the values between local, upstream and origin.

### Fast-forward

The `fast-forward` strategy updates a local package with the changes from upstream, but will
fail if the local package has been modified since it was fetched. 

### Force-delete-replace

The `force-delete-replace` strategy updates a local package with changes from upstream, but will
wipe out any modifications to the local package.

[update-doc]: /reference/cli/pkg/update/
