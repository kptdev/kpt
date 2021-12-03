A kpt package is published as a Git subdirectory containing KRM resources.
Publishing a package is just a normal Git push. This also means that any
existing Git directory of KRM resources is a valid kpt package.

As an example, let's re-publish the local `wordpress` package to your own repo.

Start by initializing the the `wordpress` directory as a Git repo if you haven't
already done so:

```shell
$ cd wordpress; git init; git add .; git commit -m "My wordpress package"
```

Tag the commit:

```shell
$ git tag v0.1
```

Push the commit which requires you to have access to the repo:

```shell
$ git push origin v0.1
```

You can then fetch the published package:

```shell
$ kpt pkg get <MY_REPO_URL>/@v0.1
```

## Monorepo Versioning

You may have a Git repo containing multiple packages. kpt provides a tagging
convention to enable packages to be independently versioned.

For example, let's assume the `wordpress` directory is not at the root of the
repo but instead is in the directory `packages/wordpress`.

Tag the commit:

```shell
$ git tag packages/wordpress/v0.1
```

Push the commit:

```shell
$ git push origin packages/wordpress/v0.1
```

You can then fetch the published package:

```shell
$ kpt pkg get <MY_REPO_URL>/packages/wordpress@v0.1
```

[tagging]: https://git-scm.com/book/en/v2/Git-Basics-Tagging
