# Consuming a package

This tutorial walks through the workflow of consuming a package.

1. Fetch a package from a remote git repository to your local filesystem
   - Any git subdirectory containing resource configuration will do
2. View its contents
3. Customize it by modifying resource field values
4. Apply it to a cluster

## Fetch a copy of a package

Fetch a copy of a package from a git repository and write it to a local directory.

```sh
export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git
kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.1.0 helloworld
```

View the package files.

```sh
tree helloworld
```

Output:

```sh
helloworld
├── Kptfile
├── README.md
├── deploy.yaml
└── service.yaml

0 directories, 4 files
```

The package contains 2 resource configuration files -- `deploy.yaml` and `service.yaml`.
These are the same types of resource configuration that would be applied with `kubectl apply`

## Show the package origin

Packages are simply git subdirectories containing resource configuration files.  Any git
subdirectory containing resource configuration files may be fetched as a package.

Show the git repo source of a package copy.

```sh
kpt pkg desc helloworld
```

Output:

```sh
helloworld/Kptfile
   PACKAGE NAME        DIR                       REMOTE                              REMOTE PATH              REMOTE REF   REMOTE COMMIT  
  hello-world-set   helloworld   https://github.com/GoogleContainerTools/kpt   /package-examples/helloworld-set   v0.1.0       5c1c019  
```

| Column         | Description                                           |
|----------------|-------------------------------------------------------|
| PACKAGE NAME   | published name of the package                         |
| DIR            | local directory containing the copy of the package    |
| REMOTE         | remote git repo the package was copied from           |
| REMOTE PATH    | remote repo subdirectory the package was copied from  |
| REMOTE REF     | remote repo ref the package was copied at             |
| REMOTE COMMIT  | remote repo commit the package was copied at          |


## View the package contents

Print the raw package resource configuration.

```sh
kpt cfg cat helloworld
```

Or print a condensed view.

```sh
kpt cfg tree helloworld
```

Output:

```sh
helloworld
├── [deploy.yaml]  Deployment helloworld-gke
└── [service.yaml]  Service helloworld-gke
```


## Customize a package

Packaged resource configuration may be directly modified using tools such as
text editors.

Additionally, packages may publish custom per-object field *setters* which
enable specific resource fields to be modified programatically from the
command line.

List the available setters.

```sh
kpt cfg list-setters helloworld
```

Output:

```sh
    NAME            DESCRIPTION         VALUE    TYPE     COUNT   SETBY  
  http-port   'helloworld port'         80      integer   3              
  image-tag   'hello-world image tag'   0.1.0   string    1              
  replicas    'helloworld replicas'     5       integer   1 
```

This package contains three setters which may be used to set resource configuration
fields through kpt: `http-port`, `image-tag` and `replicas`.

```sh
# Change http-port to 8080
kpt cfg set helloworld http-port 8080

# Change the image tag to 0.1.1
kpt cfg set helloworld image-tag 0.1.1 

# Change the replicas to 3
kpt cfg set helloworld replicas 3 
```

View the setter values after the change.

```sh
kpt cfg list-setters helloworld
```

Output:

```sh
    NAME            DESCRIPTION         VALUE    TYPE     COUNT   SETBY  
  http-port   'helloworld port'         8080    integer   3              
  image-tag   'hello-world image tag'   0.1.1   string    1              
  replicas    'helloworld replicas'     3       integer   1
```

Package consumers may add their own setters to the package.

```sh
# add a setter for the Service name 
# - the package directory is 'helloworld'
# - the setter name to create is 'service-name'
# - the current value of the setter is 'helloworld-gke' -- this must match the actual field value currently
# - the type of the value is a 'string'
# - the name of the field that will be set is 'name'
# - the kind of resource to create the setter for is 'Service' (optional)
# - the description of the field is 'the service name' (optional)
kpt cfg create-setter helloworld service-name helloworld-gke  \
    --type "string" --field name --kind Service --description "the service name"
```

List the setters again -- we should see the new one.

```
$ kpt cfg list-setters helloworld
    NAME              DESCRIPTION             VALUE         TYPE     COUNT   SETBY  
  http-port       'helloworld port'         8080             integer   3              
  image-tag       'hello-world image tag'   0.1.1            string    1              
  replicas        'helloworld replicas'     3                integer   1              
  service-name   'the service name'        helloworld-gke   string    1  
```

The `service-name` setter has been added, with the current value of `helloworld-gke`.

## Apply a package

The resource configuration in the package can be applied to a cluster using `kubectl apply`.

```sh
kubectl apply -R -f helloworld
```

Output:

```sh
deployment.apps/helloworld-gke created
service/helloworld-gke created
```
