## kpt sync

Sync package dependencies locally *declared* in a package Kptfile.

### Synopsis

Sync package dependencies locally *declared* in a package Kptfile.

For each *dependency* in a Kptfile, ensure that it exists locally with the
matching *repo* and *ref*.

    kpt sync LOCAL_PKG_DIR [flags]

  LOCAL_PKG_DIR:
  
    Local package with dependencies to sync.  Directory must exist and contain a Kptfile.

#### Env Vars

  KPT_CACHE_DIR:
  
    Controls where to cache remote packages during updates.
    Defaults to ~/.kpt/repos/
    
#### Dependencies
    
Dependencies are specified in the `Kptfile` `dependencies` field.  e.g.

    apiVersion: kpt.dev/v1alpha1
    kind: Kptfile
    dependencies:
    - name: cockroachdb-storage
      path: local/destination/dir
      git:
        repo: "https://github.com/pwittrock/examples"
        directory: "staging/cockroachdb"
        ref: "v1.0.0"


Dependencies have following schema:

    name: <user specified name>
    path: <local path (relative to the Kptfile) to fetch the dependency to>
    git:
      repo: <git repository>
      directory: <sub-directory under the git repository>
      ref: <git reference -- e.g. tag, branch, commit, etc>
    updateStrategy: <strategy to use when updating the dependency -- see kpt help update for more details>
    ensureNotExists: <remove the dependency, mutually exclusive with git>

Dependencies maybe be updated by updating their `git.ref` field and running `kpt sync`
against the directory.

### Examples

  Example Kptfile to sync:

    # file: my-package-dir/Kptfile

    apiVersion: kpt.dev/v1alpha1
    kind: Kptfile
    # list of dependencies to sync
    dependencies:
    - name: cockroachdb-storage
      # fetch the remote dependency to this local dir
      path: local/destination/dir
      git:
        # repo is the git respository
        repo: "https://github.com/pwittrock/examples"
        # directory is the git subdirectory
        directory: "staging/cockroachdb"
        # ref is the ref to fetch
        ref: "v1.0.0"
    - name: app1
      path: local/destination/dir1
      git:
        repo: "https://github.com/pwittrock/examples"
        directory: "staging/javaee"
        ref: "v1.0.0"
      # set the strategy for applying package updates
      updateStrategy: "resource-merge"
    - name: app2
      path: local/destination/dir2
      # declaratively delete this dependency
      ensureNotExists: true

  Example invocation:

    # print the dependencies that would be modified
    kpt sync my-package-dir/ --dry-run

    # sync the dependencies
    kpt sync my-package-dir/
