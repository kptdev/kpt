.
==================================================

# NAME

  hello world set

# SYNOPSIS

    kpt pkg get 

# Description

  Sample hello world package using custom setters for customization.


  This package was created to run the kubernetes-engine-samples
  [quickstart application](https://github.com/GoogleCloudPlatform/kubernetes-engine-samples/tree/master/quickstart).

  It exposes several custom *setters* for customizing the configuration after it has been
  fetched.

  List the package contents:

    $ kustomize config tree . --image --replicas --ports
    helloworld-set
    ├── [deploy.yaml]  Deployment helloworld-gke
    │   ├── spec.replicas: 1
    │   └── spec.template.spec.containers
    │       └── 0
    │           ├── image: gcr.io/kpt-dev/helloworld-gke:0.1.0
    │           └── ports: [{name: http, containerPort: 80}]
    └── [service.yaml]  Service helloworld-gke
        └── spec.ports: [{protocol: TCP, port: 80, targetPort: http}]

  List the package setters:

    $ kustomize config set helloworld-set/
        NAME            DESCRIPTION         VALUE    TYPE     COUNT   SETBY
      http-port   'helloworld port'         80      integer   3
      image-tag   'hello-world image tag'   0.1.0   string    1
      replicas    'helloworld replicas'     1       string    1

  Set a field:

    $ kustomize config set helloworld-set/ replicas 5
    set 1 fields

  View the changes:

    $ kustomize config tree helloworld-set/  --replicas
    helloworld-set
    ├── [deploy.yaml]  Deployment helloworld-gke
    │   └── spec.replicas: 5
    └── [service.yaml]  Service helloworld-gke

# SEE ALSO

