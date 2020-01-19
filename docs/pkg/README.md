## pkg

Fetch, update and sync packages using git

![alt text][demo]

[image-script]

Commands: [desc], [diff], [get], [init], [man], [sync], [update]

### Synopsis

Packages are collections of Resource configuration stored in git repositories.
They may be an entire repo, or a subdirectory within a repo.

| Command  | Description                             |
|----------|-----------------------------------------|
| [desc]   | print package origin                    |
| [diff]   | diff a local package against upstream   |
| [get]    | fetching packages from git repos        |
| [init]   | initialize an empty package             |
| [man]    | print package documentation             |
| [sync]   | fetch and update packages declaratively |
| [update] | applying upstream package updates       |

#### Package Format

1. **Any git repository containing Resource configuration may be used as a package**, no
   additional structure or formatting is necessary.
2. **Any package may be applied with `kubectl apply -R -f`** or `kpt svr apply -R -f`.
3. Packages **may be customized in place either manually with (e.g. `vi`) or programmatically**.

#### Example imperative package workflow

1. [kpt pkg get](get.md) to get a package
2. [kpt cfg set](../cfg/set.md), [kpt fn run](../fn/run.md) or `vi` to modify configuration
3. `git add` && `git commit`
4. `kubectl apply` or [kpt svr apply](../svr/apply.md) to a cluster: 
5. [kpt pkg update](update.md) to pull in new changes
6. `kubectl apply` or [kpt svr apply](../svr/apply.md) to a cluster

#### Example declarative package workflow

1. [kpt pkg init](init.md)
2. [kpt pkg sync set](sync-set.md) dev version of a package
3. [kpt pkg sync set](sync-set.md) prod version of a package
4. `git add` && `git commit`
5. [kpt svr apply --context=dev](../svr/apply.md) or `kubectl apply --context dev` apply to dev
6. [kpt svr apply --context=prod](../svr/apply.md) or or `kubectl apply --context prod` apply to prod

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

### Also See Command Groups

[cfg], [fn]

### 

[apply]: ../svr/apply.md
[cfg]: ../cfg/README.md
[demo]: https://storage.googleapis.com/kpt-dev/docs/pkg.gif "kpt pkg"
[desc]: desc.md
[diff]: diff.md
[fn]: ../fn/README.md
[functions]: ../fn/README.md
[get]: get.md
[image-script]: ../../gifs/pkg.sh
[init]: init.md
[man]: man.md
[setters]: ../cfg/set.md
[sync]: sync.md
[update]: update.md
