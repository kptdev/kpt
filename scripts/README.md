# Scripts

Helper scripts for kpt repository.

- Generates `LICENSES.txt` file for kpt release, which includes the
  licenses of the transitive dependencies of kpt.
- Generates `lib.zip`, which is a package of source files necessary
  to satisfy the Mozilla license.

## Generating LICENSES.txt and source code package

From top-level kpt directory:

```shell
./scripts/create-licenses.sh
```

This script will generate files `LICENSES.txt` and `lib.zip` in the
top-level directory. These files will eventually be included in the kpt
release tarball. This script will first run `go mod vendor` to
generate the vendored dependencies. The script removes the contents
of the vendor directory upon success.
