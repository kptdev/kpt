# How to consume a package #

Consuming a package means that you fetch a package from a remote git repository to your local filesystem,
customize it by changing values of the resource fields that your are interested in and finally apply it to
your cluster.

This tutorial walks you through the workflow of consuming a package with an example `helloworld`.

## Get a package

`kpt get pkg` gets a package from a git repository
```
$ export SRC_REPO=git@github.com:GoogleContainerTools/kpt.git
$ kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.1.0 helloworld
```

## View a package
To view the description of a package, run `kpt pkg desc`.
```
$ kpt pkg desc helloworld
helloworld/Kptfile
   PACKAGE NAME        DIR                       REMOTE                              REMOTE PATH              REMOTE REF   REMOTE COMMIT  
  hello-world-set   helloworld   git@github.com:GoogleContainerTools/kpt   /package-examples/helloworld-set   v0.1.0       5c1c019  
```

The tree structure of the package can be viewed by `kpt cfg tree`
```
$ kpt cfg tree helloworld
helloworld
├── [deploy.yaml]  Deployment helloworld-gke
└── [service.yaml]  Service helloworld-gke
```

If you want to take a look at the raw K8s resources, run `kpt cfg cat`
```
$ kpt cfg cat helloworld
```

## Customize a package
A package contains a list of setters that allow users to customize the package.

List all the setters by kpt cfg list-setters
```
$ kpt cfg list-setters helloworld
    NAME            DESCRIPTION         VALUE    TYPE     COUNT   SETBY  
  http-port   'helloworld port'         80      integer   3              
  image-tag   'hello-world image tag'   0.1.0   string    1              
  replicas    'helloworld replicas'     5       integer   1 
```

It contains three setters: `http-port`, `image-tag` and `replicas`.
To update the value of any setter, run `kpt cfg set`.
```
# Change http-port to 8080
$ kpt cfg set helloworld http-port 8080

# Change the image tag to 0.1.1
$ kpt cfg set helloworld image-tag 0.1.1 

# Change the replicas to 3
$ kpt cfg set helloworld replicas 3 
```

View the setter values after the change.
```
$ kpt cfg list-setters helloworld
    NAME            DESCRIPTION         VALUE    TYPE     COUNT   SETBY  
  http-port   'helloworld port'         8080    integer   3              
  image-tag   'hello-world image tag'   0.1.1   string    1              
  replicas    'helloworld replicas'     3       integer   1
```

If you need to customize other field which is not included in the existing setter list.
You can add a setter for it by `kpt cfg create-setter`.

```
# Add a setter for the Service name 
$ kpt cfg create-setter helloworld service-name helloworld-gke  --type "string" --field name --kind Service --description "the service name"
```

Here the `service-name` is the new setter name and `helloworld-gke` is the setter value.
Note that when creating the setter, the setter value must match the existing field value from the package.

List the setters again. You can see `service-name` is in the setter list.

```
$ kpt cfg list-setters helloworld
    NAME              DESCRIPTION             VALUE         TYPE     COUNT   SETBY  
  http-port       'helloworld port'         8080             integer   3              
  image-tag       'hello-world image tag'   0.1.1            string    1              
  replicas        'helloworld replicas'     3                integer   1              
  servicei-name   'the service name'        helloworld-gke   string    1  
```

## Apply a package
The package can be applied to a cluster by `kubectl apply`.
```
$ kubectl apply -R -f helloworld
deployment.apps/helloworld-gke created
service/helloworld-gke created
```
