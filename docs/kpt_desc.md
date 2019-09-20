## kpt desc

Display package description

### Synopsis

Display package description.

Desc reads package information in given DIRs and displays it in tabular format.
Input can be a list of package directories (defaults to the current directory if not specifed).
Directory with a Kptfile is considered to be a valid package.


```
kpt desc [DIR]... [flags]
```

### Examples

```
	# display description for package in current directory
	kpt desc

	# display description for packages in directories with 'prod-' prefix
	kpt desc prod-*

```

### Options

```
  -h, --help   help for desc
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

