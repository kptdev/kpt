## desc

Display package descriptions

### Synopsis

Display package descriptions.

    kpt pkg desc [DIR...]

`desc` reads package information in given DIRs and displays it in tabular format.
Input can be a list of package directories (defaults to the current directory if not specifed).
Any directory with a Kptfile is considered to be a package.

    kpt pkg desc [DIR]...

### Examples

    # display description for package in current directory
    kpt pkg desc

    # display description for packages in directories with 'prod-' prefix
    kpt pkg desc prod-*

