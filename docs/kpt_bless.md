## kpt bless

Initialize suggested package meta for a local config directory

### Synopsis

Initialize suggested package meta for a local config directory.

Any directory containing Kubernetes Resource Configuration may be treated as
remote package without the existence of additional packaging metadata.

* Resource Configuration may be placed anywhere under DIR as *.yaml files.
* DIR may contain additional non-Resource Configuration files.
* DIR must be pushed to a git repo or repo subdirectory.

Bless will augment an existing local directory with packaging metadata to help
with discovery.

Bless will:

* Create a Kptfile with package name and metadata if it doesn't exist
* Create a Man.md for package documentation if it doesn't exist

Args:

  DIR:
    Defaults to '.'
    Bless fails if DIR does not exist

```
kpt bless DIR [flags]
```

### Examples

```

	# writes suggested package meta if not found
	kpt bless ./ --tag kpt.dev/app=cockroachdb --description "my cockroachdb implementation"
```

### Options

```
      --description string   short description of the package. (default "sample description")
  -h, --help                 help for bless
      --name string          package name.  defaults to the directory base name.
      --tag strings          list of tags for the package.
      --url string           link to page with information about the package.
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

