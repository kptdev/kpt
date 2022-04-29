# Git Repository Test Data

The `tar` files in this directory contain archived Git repositories
which the tests unarchive into a temporary directory as a starting point
of the tests. This is to avoid constructing the repositories in code by
synthesizing commits, and (over time) to set up scenarios for handling
edge cases, etc.

The repositories are archived as bare, i.e. only their `.git` directory
is included in the archive.

To inspect the git repository contents

```sh
mkdir repo
cd repo

# This will unarchive the .git directory.
tar xfv <archived-repository>.tar
# Checkout the main branch and populate the worktree.
git reset --hard main
```

The scripts used to create and archive these repositories are:
* [scripts/create-test-repo.sh](../../../../scripts/create-test-repo.sh)
* [scripts/tar-test-repo.sh](../../../../scripts/tar-test-repo.sh)
