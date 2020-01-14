## pkg

Fetch, update and sync packages using git.

![alt text][demo]

### Synopsis

`pkg` manages Resource configuration packages.

Packages are collections of Resource configuration stored in git repositories.
They may be either the entire repo, or only a subdirectory of the repo.

**Any git repository containing Resource configuration may be used as a package**,
no additional structure or formatting is necessary for kpt to be able to fetch or
pull updates from the package.

Packages may be customized using various techniques such as [setters] and [functions].
Packages may be applied to a cluster using [apply].

A typical package workflow:

1. `kpt pkg get` to get a package
2. `kpt config set`, `kpt config run` or `vi` to modify configuration
3. `git add && git commit` to save package
4. `kpt http apply` to a cluster
5. `kpt pkg update` to pull in new changes

A collection of packages to fetch and update may be specified declaratively using `kpt sync`.

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
    fetching package /package-examples/helloworld-set from \
        git@github.com:GoogleContainerTools/kpt to helloworld

    # pull in upstream updates by merging Resources
    $ kpt pkg update helloworld@v0.2.0 --strategy=resource-merge
    updating package helloworld to v0.2.0

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

[demo]: ../gifs/pkg.gif "Five Minute Demo"
[setters]: ../config/setters.md
[functions]: ../functions
[apply]: ../http/apply.md