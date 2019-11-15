## desc

Display package descriptions

### Synopsis

Display package descriptions.

`desc` reads package information in given DIRs and displays it in tabular format.
Input can be a list of package directories (defaults to the current directory if not specifed).
Any directory with a Kptfile is considered to be a package.

    kpt desc [DIR]... [flags]

### Examples

```
# display description for package in current directory
kpt desc

# display description for packages in directories with 'prod-' prefix
kpt desc prod-*
```
