Packaging in kpt is based on Git forking. Producers publish packages by
committing them to a Git repository. Consumers fork the package to use it.

Let's revisit the Wordpress example:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress@v0.9
```

A package in a Git repo can be fetched by specifying a branch, tag, or commit
SHA. In this case, we are specifying tag `v0.9`.

?> Refer to the [get command reference][get-doc] for usage.

The `Kptfile` contains metadata about the origin of the forked package. Take a
look at the content of the `Kptfile` on your local filesystem:

```yaml
# wordpress/Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: wordpress
upstream:
  type: git
  git:
    repo: https://github.com/GoogleContainerTools/kpt
    directory: /package-examples/wordpress
    ref: v0.9
  updateStrategy: resource-merge
upstreamLock:
  type: git
  git:
    repo: https://github.com/GoogleContainerTools/kpt
    directory: /package-examples/wordpress
    ref: package-examples/wordpress/v0.9
    commit: b9ea0bca019dafa9f9f91fd428385597c708518c
info:
  emails:
    - kpt-team@google.com
  description: This is an example wordpress package with mysql subpackage.
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
        app: wordpress
  validators:
    - image: gcr.io/kpt-fn/kubeval:v0.3
```

The `Kptfile` contains two sections to keep track of the upstream package:

1. The `upstream` section contains the user-specified Git reference to the
   upstream package. This contains three pieces of information:
   - `repo`: The Git repository where the package can be found
   - `directory`: The directory within the Git repository where this package can
     be found
   - `ref`: The Git reference for the package. This can be either a branch, tag,
     or commit SHA.
2. The `upstreamLock` section records the upstream Git reference (exact Git SHA)
   that was fetched by kpt. This section is managed by kpt and should not be
   changed manually.

Now, let's look at the `Kptfile` for the `mysql` subpackage:

```yaml
# wordpress/mysql/Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: mysql
info:
  emails:
    - kpt-team@google.com
  description: This is an example mysql package.
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
        tier: mysql
```

As you can see, this `Kptfile` doesn't have the `upstream` and `upstreamLock`
sections. This is because there are two different package types in kpt:

- **Independent package:** A package where the `Kptfile` has `upstream` defined.
- **Dependent package:** A package where the `Kptfile` doesn’t have `upstream`
  defined.

In this case, the `mysql` subpackage is a _dependent package_. The upstream
package for `mysql` is automatically inferred from the parent package. You can
think of the `Kptfile` in the `mysql` package as implicitly inheriting the
`upstream` section of its parent, with the only difference being that
`upstream.directory` in the subpackage would instead point to
`/package-examples/wordpress/mysql`.

## Package Name and Identifier

It is possible to specify a different local directory name to the `get` command.
For example, the following fetches the packages to a directory named
`mywordpress`:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress@v0.9 mywordpress
```

The _name of a package_ is given by its directory name. Since the Kptfile is a
KRM resource and follows the familiar structure of KRM resources, the name of
the package is also available from the `metadata.name` field. This must always
be the name of the directory, and kpt will update it automatically when forking
a package. In this case, `metadata.name` is set to `mywordpress`.

In general, the package name is not unique. The _unique identifier_ for a
package is defined as the relative path from the top package to the subpackage.
For example, we could have two subpackages with the name `mysql` having the
following identifiers:

- `wordpress/backend/mysql`
- `wordpress/frontend/mysql`

[get-doc]: /reference/cli/pkg/get/
