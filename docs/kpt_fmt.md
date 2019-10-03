## kpt fmt

Format yaml configuration files

### Synopsis

Format yaml configuration files

Fmt will format input by ordering fields and unordered list items in Kubernetes
objects.  Inputs may be directories, files or stdin, and their contents must
include both apiVersion and kind fields.

- Stdin inputs are formatted and written to stdout
- File inputs (args) are formatted and written back to the file
- Directory inputs (args) are walked, each encountered .yaml and .yml file
  acts as an input

For inputs which contain multiple yaml documents separated by \n---\n,
each document will be formatted and written back to the file in the original
order.

Field ordering roughly follows the ordering defined in the source Kubernetes
resource definitions (i.e. go structures), falling back on lexicographical
sorting for unrecognized fields.

Unordered list item ordering is defined for specific Resource types and
field paths.

- .spec.template.spec.containers (by element name)
- .webhooks.rules.operations (by element value)


```
kpt fmt [flags]
```

### Examples

```

	# format file1.yaml and file2.yml
	kpt fmt file1.yaml file2.yml

	# format all *.yaml and *.yml recursively traversing directories
	kpt fmt dir/

	# format kubectl output
	kubectl get -o yaml deployments | kpt fmt

	# format kustomize output
	kustomize build | kpt fmt

```

### Options

```
  -h, --help               help for fmt
      --keep-annotations   if true, keep index and filename annotations set on Resources.
      --pattern string     pattern to use for generating filenames for resources -- may contain the following
                           formatting substitution verbs {'%n': 'metadata.name', '%s': 'metadata.namespace', '%k': 'kind'} (default "%n_%k.yaml")
      --set-filenames      if true, set default filenames on Resources without them
```

### SEE ALSO

* [kpt](kpt.md)	 - Kpt Packaging Tool

