## man

Format and display package documentation if it exists

### Synopsis

    kpt pkg man LOCAL_PKG_DIR [flags]

  LOCAL_PKG_DIR:

    local path to a package.

If package documentation is missing from the package or 'man' is not installed,
the command will fail.

### Examples

    # display package documentation
    kpt pkg man my-package/

    # display subpackage documentation
    kpt pkg man my-package/sub-package/

