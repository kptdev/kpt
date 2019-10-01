## tutorials 3-modifying-local-packages

Use package-specific commands to modify package contents

### Synopsis

Resources in local packages may be modified using commands which are dynamically
enabled based on the package content -- e.g. the 'set image' command is available if the 
package contains a Resource with 'spec.template.spec.containers'.

Stage the package:

	kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0 cockroachdb/

## Show the set of commands available for the package

	$ kpt cockroachdb/ -h
	...
	Available Commands:
	  get         
	  set
	...

  2 subcommand groups are shown

	$ kpt cockroachdb/ get -h
	...
	Available Commands:
	  cpu-limits          Get cpu-limits for a container
	  cpu-reservations    Get cpu-reservations for a container
	  env                 Get an environment variable from a container
	  image               Get image for a container
	  memory-limits       Get memory-limits for a container
	  memory-reservations Get memory-reservations for a container
	  replicas            Get the replicas for a Resource

  This is the set of get commands enabled for the cockroachdb package


	$ kpt cockroachdb/ set -h
	...
	Available Commands:
	  cpu-limits          Set cpu-limits for a container
	  cpu-reservations    Set cpu-reservations for a container
	  env                 Set an environment variable on a container
	  image               Set the image on a container
	  memory-limits       Set memory-limits for a container
	  memory-reservations Set memory-reservations for a container
	  replicas            Set the replicas for a Resource


  This is the set of set commands enabled for the cockroachdb package


## Get and Set the image

	$ kpt tree cockroachdb/
	cockroachdb
	├── [cockroachdb-statefulset.yaml]  v1.Service cockroachdb
	├── [cockroachdb-statefulset.yaml]  apps/v1.StatefulSet cockroachdb
	├── [cockroachdb-statefulset.yaml]  policy/v1beta1.PodDisruptionBudget cockroachdb-budget
	└── [cockroachdb-statefulset.yaml]  v1.Service cockroachdb-public

  tree listed the Resources to operate against

	$ kpt cockroachdb/ get image cockroachdb
	cockroachdb/cockroach:v1.1.0

  get image printed the container image for the Resource + Container matching the name "cockroachdb"

	$ kpt cockroachdb/ set image cockroachdb --value cockroachdb/cockroach:v1.1.1
	$ kpt cockroachdb/ get image cockroachdb
	cockroachdb/cockroach:v1.1.1

  set image set the container image to a new value

## Get and Set the replicas

	$ kpt cockroachdb/ get replicas cockroachdb
	3

  get replicas printed the current replica count

	$ kpt cockroachdb/ set replicas cockroachdb --value 5
	$ kpt cockroachdb/ get replicas cockroachdb
	5

## Other commands

  Explore the rest of the commands listed by -h.


```
tutorials 3-modifying-local-packages [flags]
```

### Options

```
  -h, --help   help for 3-modifying-local-packages
```

### SEE ALSO

* [tutorials](tutorials.md)	 - Contains tutorials for using kpt

