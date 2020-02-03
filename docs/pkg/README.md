## pkg

Fetch, update, and sync configuration files using git

<link rel="stylesheet" type="text/css" href="/kpt/gifs/asciinema-player.css" />
<asciinema-player src="/kpt/gifs/pkg.cast" speed="1" theme="solarized-dark" cols="60" rows="26" font-size="medium" idle-time-limit="1"></asciinema-player>
<script src="/kpt/gifs/asciinema-player.js"></script>

    # run the tutorial from the cli
    kpt tutorial pkg

[tutorial-script]

### Synopsis

Packages are collections of resource configuration files stored in git repositories.
They may be an entire repo, or a subdirectory within a repo.

| Command  | Description                             |
|----------|-----------------------------------------|
| [desc]   | print the package origin                |
| [diff]   | diff a local package against upstream   |
| [get]    | fetch a package from a git repo         |
| [init]   | initialize an empty package             |
| [sync]   | fetch and update packages declaratively |
| [update] | apply upstream package updates          |

**Data Flow**: git repo -> kpt [pkg] -> local files or stdout

| Configuration Read From | Configuration Written To |
|-------------------------|--------------------------|
| git repository          | local files              |

#### Package Format

1. **Any git repository containing resource configuration files may be used as a package**, no
   additional structure or formatting is necessary.
2. **Any package may be applied with `kubectl apply -R -f`**.
3. Packages **may be customized in place either manually (e.g. with `vi`) or programmatically**.
4. Packages **must** be worked on within a local git repo.

![day1 workflow][day1workflow]
![dayN workflow][dayNworkflow]

#### Example imperative package workflow

1. [kpt pkg get](get.md) to get a package
2. [kpt cfg set](../cfg/set.md), [kpt fn run](../fn/run.md) or `vi` to modify configuration
3. `git add` && `git commit`
4. `kubectl apply` to a cluster:
5. [kpt pkg update](update.md) to pull in new changes
6. `kubectl apply` to a cluster

#### Example declarative package workflow

1. [kpt pkg init](init.md)
2. [kpt pkg sync set](sync-set.md) dev version of a package
3. [kpt pkg sync set](sync-set.md) prod version of a package
4. `git add` && `git commit`
5. `kubectl apply --context dev` apply to dev
6. `kubectl apply --context prod` apply to prod

#### Model

1. **Packages are simply subdirectories of resource configuration files in git**
    * They may also contain supplemental non-resource artifacts, such as markdown files, templates, etc.
    * The ability to fetch a subdirectory of a git repo is a key difference compared to 
      [git subtree](https://github.com/git/git/blob/master/contrib/subtree/git-subtree.txt).

2. **Any existing git subdirectory containing resource configuration files may be used as a package**
    * Nothing besides a git directory containing resource configuration is required.
    * e.g. any [example in the examples repo](https://github.com/kubernetes/examples/staging/cockroachdb) may
      be used as a package:

          kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb \
            my-cockroachdb
          kubectl apply -R -f my-cockroachdb

3. **Packages should use git references for versioning**.
    * We recommend package authors use semantic versioning when publishing packages for others to consume.

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

7. **Packages must be changed within a git repo**.
    * kpt facilitates configuration reuse. Key to that reuse is git. Git is used
      both as a unit to define the boundary of a git repo, but also as the
      boundary of a local workspace. In this way, any workspace is in itself a
      package. Local customizations of packages (or use of specific versions)
      can be re-published as the new canonical package for other users. kpt
      requires any customization to be committed to git before package updates
      can be reconciled.

### Examples

    # create your workspace
    $ mkdir hello-world-workspace
    $ cd hello-world-workspace
    $ git init

    # get the package
    $ export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git
    $ kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.1.0 helloworld

    # add helloworld to your workspace
    $ git add .
    $ git commit -am "Add helloworld to my workspace."

    # pull in upstream updates by merging Resources
    $ kpt pkg update helloworld@v0.2.0 --strategy=resource-merge

    # you can review the update and then commit your changes with git here

    # manage a collection of packages declaratively
    $ kpt pkg init ./ --description "my package"
    $ kpt pkg sync set $SRC_REPO/package-examples/helloworld-set@v0.1.0 \
        hello-world --strategy=resource-merge
    $ kpt pkg sync ./

    # commit your packages
    git add .
    git commit -am "Synced hello-world v1 via set."

    # update the package with sync
    $ kpt pkg sync set $SRC_REPO/package-examples/helloworld-set@v0.2.0 \
        hello-world --strategy=resource-merge
    $ kpt pkg sync ./

    # Commit again.

### Also See Command Groups

[cfg], [fn]

###
[day1workflow]: ../images/day1workflow.jpg
[dayNworkflow]: ../images/dayNworkflow.jpg
[apply]: ../svr/apply.md
[cfg]: ../cfg/README.md
[desc]: desc.md
[diff]: diff.md
[fn]: ../fn/README.md
[functions]: ../fn/README.md
[get]: get.md
[tutorial-script]: ../gifs/pkg.sh
[init]: init.md
[setters]: ../cfg/set.md
[sync]: sync.md
[update]: update.md
