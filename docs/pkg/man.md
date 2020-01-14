## man

Format and display package documentation if it exists

### Synopsis

Format and display package documentation if it exists.    If package documentation is missing
from the package or 'man' is not installed, the command will fail.

    kpt pkg man LOCAL_PKG_DIR [flags]

  LOCAL_PKG_DIR:

    local path to a package.

### Examples


    # display package documentation
    kpt pkg man my-package/

    # display subpackage documentation
    kpt pkg man my-package/sub-package/

