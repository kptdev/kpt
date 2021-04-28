Since a forked package contains a reference to its upstream package, kpt can fetch any update to the
upstream package and merge it with local changes.

The package can be updated with changes from upstream in a declarative way by updating the
`upstream` section in the `Kptfile` then running the `update` command.

For example, change the value of `upstream.git.ref` to `v0.2`:

```yaml
upstream:
  type: git
  git:
    repo: https://github.com/GoogleContainerTools/kpt
    directory: /package-examples/wordpress
    # Update v0.1 to v0.2
    ref: v0.2
  updateStrategy: resource-merge
```

and then run:

```shell
$ kpt pkg update wordpress
```

Alternatively, the user can provide the version directly to the `update` command, and kpt will take
care of first updating the `upstream` section of the `Kptfile` and then performing the merge
operation:

```shell
$ kpt pkg update wordpress@v0.2
```

The `update` command updates the local `wordpress` package and the dependent `mysql` package to the
upstream version `v0.2` by doing a 3-way merge between:

1. Original upstream commit
2. New upstream commit
3. Local (edited) package

Several different strategies are available to handle the merge. By default, the `resource-merge`
strategy is used which performs a structural comparison of the resource using OpenAPI schema.

> Refer to the [command reference][update-doc] for more details.

[update-doc]: /reference/pkg/update/

TODO(#1827): Handling merge conflicts
