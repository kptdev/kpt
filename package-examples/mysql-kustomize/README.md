# mysql-kustomize

This is a package that has several hydration functions in it's pipeline
as well as utilzing kustomize to patch an upstream configuration.  The 
upstream configuration is saved in the /upstream folder and the /instance
patch is configured using kpt functions.

For the final configuration kustomize is used to build the patched config
with `kpt live apply` doing the deployment.

## Steps

1. [Fetch the package](#fetch-the-package)
2. [View the package contents](#view-the-package-contents)
3. [Configure functions](#configure-functions)
4. [Render the declared values](#render-the-declared-values)
5. [Apply the package](#apply-the-package)

### Fetch the package

Get the example package on to local using `kpt pkg get`

  $ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/mysql-kustomize

    fetching package /package-examples/mysql-kustomize from https://github.com/GoogleContainerTools/kpt to mysql-kustomize

### View the package contents

List the package contents in a tree structure.

  $ kpt pkg tree mysql-kustomize/

    PKG: mysql-kustomize
    ├── [Kptfile]  Kptfile consumer
    ├── instance
    │   ├── [kustomization.yaml]  Kustomization local instance kustomization
    │   ├── [service.yaml]  Service none
    │   └── [statefulset.yaml]  StatefulSet mysql
    └── upstream
        ├── [kustomization.yaml]  Kustomization upstream kustomization
        ├── [service.yaml]  Service mysql
        ├── [service.yaml]  Service mysql-read
        └── [statefulset.yaml]  StatefulSet mysql

### Configure functions

The package contains a function pipeline in the `Kptfile` which has
one `apply-setters` and `set-namespace` functions.  The `apply-setters` 
function allows you to set a simple value throughout the package configuration.
The `set-namespace` function allows you to set the namespace on the resources.

  pipeline:
    mutators:
      - image: gcr.io/kpt-fn/apply-setters:unstable
        configMap:
            mysql-user: userone
            mysql-database: maindb
            skip-grant-tables: true
            cpu: 100m
            memory: 256Mi
            port: 3306
      - image: gcr.io/kpt-fn/set-namespace:unstable
        configMap:
            namespace: ns-test


### Render the declared values

Render the changes in the hydration pipeline by using `kpt fn render` command:

  $ kpt fn render mysql-kustomize/

    package "mysql-kustomize": running function "gcr.io/kpt-fn/apply-setters:unstable": SUCCESS
    package "mysql-kustomize": running function "gcr.io/kpt-fn/set-namespace:unstable": SUCCESS
    package "mysql-kustomize": rendered successfully

### Apply the package

Apply all the contents of the package using kustomize build and kubectl.

  $ kustomize build mysql-kustomize/instance | kubectl apply -f -

    configmap/mysql-md76k5d77k created
    secret/mysql-h4896b7hgh created
    service/mysql created
    statefulset.apps/mysql created