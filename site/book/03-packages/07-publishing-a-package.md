A kpt package is published as a Git subdirectory containing KRM resources. Publishing a
package is just a normal Git push. This also means that any existing Git directory of KRM
resources is a valid kpt package.

As an example, let's re-publish the local `wordpress` package to your own repo.

Start by initializing the the `wordpress` directory as a Git repo:

```shell
$ git init wordpress
```

Create a Git commit:

```shell
$ cd wordpress
$ git add . && git commit -m "Add wordpress package"
```

Tag and pushes the commit:

```shell
$ git tag v0.1
$ git push v0.1 # requires you to have an upstream repo
```

You can then fetch the published package:

```shell
$ kpt pkg get <MY_REPO_URL>/@v0.1
```

## Per-directory Versioning

You may have a Git repo containing multiple packages. kpt provides a tagging
convention to enable packages to be independently versioned.

For example, let's assume the `wordpress` directory is not at the root of the repo,
but instead is in the directory `packages/wordpress`:

```shell
$ git tag packages/wordpress/v0.1
$ git push packages/wordpress/v0.1 # requires you to have an upstream repo
```

You can then fetch the published package:

```shell
$ kpt pkg get <MY_REPO_URL>/packages/wordpress@v0.1
```

[tagging]: https://git-scm.com/book/en/v2/Git-Basics-Tagging
