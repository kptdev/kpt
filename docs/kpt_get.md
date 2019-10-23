## kpt get

Fetch a package from a git repository

### Synopsis

Fetch a package from a git repository.
Args:

  REPO_URI:
    URI of a git repository containing 1 or more packages as subdirectories.
    In most cases the .git suffix should be specified to delimit the REPO_URI from the PKG_PATH,
    but this is not required for widely recognized repo prefixes.  If get cannot parse the repo
    for the directory and version, then it will print an error asking for '.git' to be specified
    as part of the argument.
    e.g. https://github.com/kubernetes/examples.git
    Specify - to read from stdin.

  PKG_PATH:
    Path to remote subdirectory containing Kubernetes Resource configuration files or directories.
    Defaults to the root directory.
    Uses '/' as the path separator (regardless of OS).
    e.g. staging/cockroachdb

  VERSION:
    A git tag, branch, ref or commit for the remote version of the package to fetch.
    Defaults to the repository master branch.
    e.g. @master

  LOCAL_DEST_DIRECTORY:
    The local directory to fetch the package to.
    e.g. ./my-cockroachdb-copy

    * If the directory does NOT exist: create the specified directory and write the package contents to it
    * If the directory DOES exist: create a NEW directory under the specified one, defaulting the name to the Base of REPO/PKG_PATH
    * If the directory DOES exist and already contains a directory with the same name of the one that would be created: fail

```
kpt get REPO_URI[.git]/PKG_PATH[@VERSION] LOCAL_DEST_DIRECTORY [flags]
```

### Examples

```
  # fetch package cockroachdb from github.com/kubernetes/examples/staging/cockroachdb
  # creates directory ./cockroachdb/ containing the package contents
  kpt get https://github.com/kubernetes/examples.git/staging/cockroachdb@master ./

  # fetch a cockroachdb
  # if ./my-package doesn't exist, creates directory ./my-package/ containing the package contents
  kpt get https://github.com/kubernetes/examples.git/staging/cockroachdb@master ./my-package/

  # fetch package examples from github.com/kubernetes/examples
  # creates directory ./examples fetched from the provided commit
  kpt get https://github.com/kubernetes/examples.git/@8186bef8e5c0621bf80fa8106bd595aae8b62884 ./
```

### Options

```
  -h, --help             help for get
      --pattern string   Pattern to use for writing files.  
                         May contain the following formatting verbs
                         %n: metadata.name, %s: metadata.namespace, %k: kind
                          (default "%n_%k.yaml")
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

