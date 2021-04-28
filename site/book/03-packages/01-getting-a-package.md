Packaging in kpt based on Git forking. Packages are published by committing them to a Git repository
and packages are consumed by creating a new fork of the package.

Let's revisit the Wordpress example:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress@v0.1
```

A package on a Git repo can be fetched by specifying a branch, tag, or a commit SHA. In this case,
we are specifying tag `v0.1`.

> Refer to the [command reference][get-doc] for more details.

The `Kptfile` contains metadata about the origin of the forked package. Take a look at the content
of the `Kptfile` on your local filesystem:

```shell
$ cat wordpress/Kptfile
```

```yaml
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: wordpress
upstream:
  type: git
  git:
    repo: https://github.com/GoogleContainerTools/kpt
    directory: /package-examples/wordpress
    ref: v0.1
  updateStrategy: resource-merge
upstreamLock:
  type: git
  gitLock:
    repo: https://github.com/GoogleContainerTools/kpt
    directory: /package-examples/wordpress
    ref: package-examples/wordpress/v0.1
    commit: e0e0b3642969c2d14fe1d38d9698a73f18aa848f
info:
  emails:
    - kpt-team@google.com
  description: This is an example wordpress package with mysql subpackage
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configMap:
        wp-image: wordpress
        wp-tag: 4.8-apache
```

The `Kptfile` contains two sections to keep track of the upstream package:

1. The `upstream` section contains the user-specified reference to the upstream package. This
   contains three pieces of information:
   - `repo`: The git repository where the package can be found
   - `directory`: The directory within the git repository where this package can be found
   - `ref`: The git reference for the package. This can be either a brach, tag, or commit SHA.
2. The `upstreamLock` section contains the resolved upstream reference. This section is managed by
   kpt and should not be changed manually.

So essentially, the `upstream` section defines the "desired state" while the `upstreamLock` section
defines the “current state”.

Now, let's look at the `Kptfile` for the `mysql` subpackage:

```shell
$ cat wordpress/mysql/Kptfile
```

```yaml
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: mysql
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configMap:
        ms-image: mysql
        ms-tag: 5.6
```

As you can see, this `Kptfile` doesn't have the `upstream` and `upstreamLock` sections.
This is because there are two different package types in kpt:

- **Independent package:** A package where the `Kptfile` has `upstream` defined.
- **Dependent package:** A package where the `Kptfile` doesn’t have `upstream` defined.

In this case, the `mysql` subpackage is a _dependent package_. The upstream package for `mysql` is
automatically inferred from the parent package. You can think of the `Kptfile` in the `mysql`
package as implicitly inheriting the `upstream` section of its parent, with the only difference
being that `upstream.directory` points to `/package-examples/wordpress/mysql`.

## Package Name and Identifier

It is possible to specify a different local directory name to the `get` command. For example,
the following fetches the packages to a directory names `mywordpress`:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress@v0.1 mywordpress
```

The _name_ of a package is given by its directory name. Since the Kptfile is a KRM resource and
follows the familiar structure of KRM resources, the name of the package is also available from the
`metadata.name` field. This must always be the same of the directory name and kpt will update it
automatically when forking a package. In this case, `metadata.name` is set `mywordpress`.

The name of a package is unique within its parent package, but it may not be unique in a deeply
nested package hierarchy (i.e. Depth > 2). The _unique identifier_ for a package is defined as the
relative path from the top package to the subpackage. For example, we could have two subpackages
with the name `mysql` having the following identifiers:

- `wordpress/backend/mysql`
- `wordpress/frontend/mysql`

[get-doc]: /reference/pkg/get/
