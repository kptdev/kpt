## kpt duck-typed

Duck-typed commands are enabled for packages based off the package's content

### Synopsis

Duck-typed commands are enabled for packages based off the package's content.

To see the list of duck-typed and custom commands for a package, provide the package as the
first argument to kpt.

	kpt pkg/ -h

Each package may contain Resources which have commands specific to that Resource -- such
as for getting and setting fields.

Duck-typed commands are enabled for packages by inspecting the Resources in the package,
and identifying which commands may be applied to those Resources.

Commands may be enabled by the presence of specific fields in Resources -- e.g. 'set replicas'
or by the presence of specific Resources types in the package.


### Examples

```
	# list the commands for a package
	kpt PKG_NAME/ -h
	
	# get help for a specific package subcommand
	kpt PKG_NAME/ set image -h

```

### Options

```
  -h, --help   help for duck-typed
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool
* [kpt duck-typed ](kpt_duck-typed_.md)	 - 

