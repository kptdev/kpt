## pkg

Fetch, update and sync packages using git

![alt text][demo]

### Synopsis

Packages are collections of Resource configuration stored in git repositories.
They may be an entire repo, or a subdirectory within a repo.

- **Any git repository containing Resource configuration may be used as a package**,
  no additional structure or formatting is necessary.
- **Any `kpt` package may be applied with `kubectl apply -R -f`** or managed with
  any tools that operate against Resource configuration.
- Fetched packages **may be customized in place either manually with (e.g. `vi`) or
  programmatically** (e.g. [setters], [functions]).

Example package workflow:

1. `kpt pkg get` to get a package
2. `kpt cfg set`, `kpt cfg run` or `vi` to modify configuration
3. `git add` && `git commit`
4. `kubectl apply -R -f` or `kpt svr apply` to a cluster
5. `kpt pkg update` to pull in new changes
6. `kubectl apply -R -f` or `kpt svr apply` to a cluster

An alternative workflow is to use `kpt pkg sync` to specify packages to fetch and update in
declarative files.

#### Architecture

1. **Packages artifacts are Resource configuration** (rather than DSLs, templates, etc)
    * They may also contain supplemental non-Resource artifacts (e.g. README.md, Templates, etc).

2.  **Any existing git subdirectory containing Resource configuration** may be used as a package.
    * Nothing besides a git directory containing Resource configuration is required.
    * e.g. the [examples repo](https://github.com/kubernetes/examples/staging/cockroachdb) may
      be used as a package:

          kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb \
            my-cockroachdb
          kubectl apply -R -f my-cockroachdb

3. **Packages should use git references for versioning**.
    * Package authors should use semantic versioning when publishing packages.

          kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb@VERSION \
            my-cockroachdb
          kubectl apply -R -f my-cockroachdb

4. **Packages may be modified or customized in place**.
    * It is possible to directly modify the fetched package and merge upstream updates.

5. **The same package may be fetched multiple times** to separate locations.
    * Each instance may be modified and updated independently of the others.

          # fetch an instance of a java package
          kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb db1
          # make changes...

          # fetch a second instance of a java package
          kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb db2
          # make changes...

6. **Packages may pull upstream updates after they have been fetched and modified**.
    * Specify the target version to update to, and an (optional) update strategy for how to apply the
      upstream changes.

          kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb \
            my-cockroachdb
          # make changes...
          kpt pkg update my-cockroachdb@NEW_VERSION --strategy=resource-merge

### Primary Commands

**[get](get.md)**:
- fetching packages from subdirectories stored in git to local copies

**[update](update.md)**:
- applying upstream package updates to a local copy

**[init](init.md)**:
- initialize an empty package

**[sync](sync.md)**:
- defining packages to sync from remote sources using a declarative file which
  maps remote packages (repo + path + version) to local directories

### Additional Commands

**[diff](diff.md)**:
- diff a locally modified package against upstream

**[desc](desc.md)**:
- print package origin

**[man](man.md)**:
- print package documentation

### Examples

    # get the package
    export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git
    $ kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.1.0 helloworld

    # pull in upstream updates by merging Resources
    $ kpt pkg update helloworld@v0.2.0 --strategy=resource-merge

    # manage a collection of packages declaratively
    $ kpt pkg init ./ --description "my package"
    $ kpt pkg sync set $SRC_REPO/package-examples/helloworld-set@v0.1.0 \
        hello-world --strategy=resource-merge
    $ kpt pkg sync ./

    # update the package with sync
    $ kpt pkg sync set $SRC_REPO/package-examples/helloworld-set@v0.2.0 \
        hello-world --strategy=resource-merge
    $ kpt pkg sync ./

### 

[demo]: https://storage.googleapis.com/kpt-dev/docs/pkg.gif "kpt pkg"
[setters]: ../cfg/set.md
[functions]: ../fn/README.md
[apply]: ../svr/apply.md