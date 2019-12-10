## sub-check

Check for unfulfilled package substitutions

### Synopsis

Check for unfulfilled package substitutions.  Unfulfilled substitutions are substitutions
where the substitution has not yet been performed, and the *marker* is still in place.

`sub-check` looks for possible value substitutions in a package by reading the Kptfile
and exists non-0 if any of the substitutions have not been performed.

By default, `sub-check` will look for any substitutions that have not be fulfilled, but
may accept specific substitutions to check.

To print the available substitutions for a package, run `sub` on the package directory
and they will be listed as sub commands.

  PKG_DIR

    A directory containing a Kptfile with substitutions specified.

  SUBSTITUTION_NAME

    Optional name of the substitution to check.  Available substitutions names will
    be listed when running `sub` against the PKG_DIR with no other arguments.

See: `kpt sub` for more details

### Examples

    # print the unfulfilled substitutions (exits non-0)
    $ kpt sub-check my-package/
    SubstitutionName         Count D
    port                     4
    name-prefix              1
 
    # print the unfulfilled port substitutions (exits non-0)
    $ kpt sub-check my-package/ port
    NAME         COUNT
    port          4
    name-prefix   1

    # perform substitutions and then print the unfulfilled
    # port substitutions (exits 0)
    $ kpt sub my-package/ port 8080
    $ kpt sub my-package/ name-prefix prefix-
    $ kpt sub-check my-package/
    NAME         COUNT

