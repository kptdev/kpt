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
* [kpt duck-typed get-cpu-limits](kpt_duck-typed_get-cpu-limits.md)	 - Get cpu-limits for a container
* [kpt duck-typed get-cpu-requests](kpt_duck-typed_get-cpu-requests.md)	 - Get cpu-requests for a container
* [kpt duck-typed get-env](kpt_duck-typed_get-env.md)	 - Get an environment variable from a container
* [kpt duck-typed get-image](kpt_duck-typed_get-image.md)	 - Get image for a container
* [kpt duck-typed get-memory-limits](kpt_duck-typed_get-memory-limits.md)	 - Get memory-limits for a container
* [kpt duck-typed get-memory-requests](kpt_duck-typed_get-memory-requests.md)	 - Get memory-requests for a container
* [kpt duck-typed get-replicas](kpt_duck-typed_get-replicas.md)	 - Get the replicas for a Resource
* [kpt duck-typed set-cpu-limits](kpt_duck-typed_set-cpu-limits.md)	 - Set cpu-limits for a container
* [kpt duck-typed set-cpu-requests](kpt_duck-typed_set-cpu-requests.md)	 - Set cpu-requests for a container
* [kpt duck-typed set-env](kpt_duck-typed_set-env.md)	 - Set an environment variable on a container
* [kpt duck-typed set-image](kpt_duck-typed_set-image.md)	 - Set the image on a container
* [kpt duck-typed set-memory-limits](kpt_duck-typed_set-memory-limits.md)	 - Set memory-limits for a container
* [kpt duck-typed set-memory-requests](kpt_duck-typed_set-memory-requests.md)	 - Set memory-requests for a container
* [kpt duck-typed set-replicas](kpt_duck-typed_set-replicas.md)	 - Set the replicas for a Resource

