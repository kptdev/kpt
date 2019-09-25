## apis package-structure



### Synopsis

Description:
  kpt packages may be published as:

    * git repositories
    * git repository subdirectories

  kpt packages are packages of Resource Configuration as yaml files.
  As such they SHOULD contain at least one of:
  
    * Kubernetes Resource Configuration files (.yaml or .yml)
    * Kustomization.yaml
    * Subdirectories containing either of the above

 kpt packages MAY additionally contain:

    * Kptfile: package metadata (see 'kpt help kptfile')
    * MAN.md: package documentation (md2man format)
    * LICENSE: package LICENSE
    * Other kpt subpackages
    * Arbitrary files

  A configuration directory may be blessed with recommended kpt package metadata
  files using 'kpt bless dir/'

Examples:

  # * 1 resource per-file
  # * flat structure
  
  $ tree cockroachdb
  cockroachdb/
  ├── Kptfile
  ├── MAN.md
  ├── cockroachdb-pod-disruption-budget.yaml
  ├── cockroachdb-public-service.yaml
  ├── cockroachdb-service.yaml
  └── cockroachdb-statefulset.yaml

  # * multiple resources per-file
  # * nested structure
  # * contains subpackage

  $ tree wordpress/
  wordpress/
  ├── Kptfile
  ├── MAN.md
  ├── Kustomization.yaml
  ├── mysql
  │   ├── Kptfile
  │   └── mysql.yaml
  └── wordpress
      └── wordpress.yaml

```
apis package-structure [flags]
```

### Options

```
  -h, --help   help for package-structure
```

### SEE ALSO

* [apis](apis.md)	 - Contains api information for kpt

