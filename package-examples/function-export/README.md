# function-export

This is a package that has both a setter and a validation function.  It shows
how you can use gatekeeper to perform checks that are run during rendering.
Early feedback minimizes the cost of doing the fix and feedback during 
design time is the easiest one to address.

## Steps

This is a simple workflow where you can download, configure, render,
validate and apply the package:

1. [Fetch the package](#fetch-the-package)
2. [View the package contents](#view-the-package-contents)
3. [Configure functions](#configure-functions)
4. [Render the declared values](#render-the-declared-values)
5. [Apply the package](#apply-the-package)

### Fetch the package

Get the example package on to local using `kpt pkg get`

  $ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/function-export

    fetching package /package-examples/function-export from https://github.com/GoogleContainerTools/kpt to function-export

### View the package contents

List the package contents in a tree structure.

  $ kpt pkg tree function-export/

    PKG: function-export
    ├── [Kptfile]  Kptfile kpt-function-export-example
    └── resources
        ├── [resources.yaml]  Namespace development
        ├── [resources.yaml]  Deployment development/nginx-deployment
        └── constraints
            ├── [deployment-must-have-owner.yaml]  K8sRequiredLabels deployment-must-have-owner
            └── [requiredlabels.yaml]  ConstraintTemplate k8srequiredlabels

### Configure functions

The package contains a function pipeline in the `Kptfile` which has
one `apply-setter` and `gatekeeper-validate` functions.  
The `apply-setter` function allows you to set a simple value throughout the 
package configuration.  In this case it's namespace label.  The
`gatekeeper-validate` function allows you to use gatekeeper for checks on
the configuration.

  pipeline:
    mutators:
      - image: gcr.io/kpt-functions/label-namespace
        configMap:
          label_name: color
          label_value: blue      
    validators:
      - image: gcr.io/kpt-functions/gatekeeper-validate



### Render the declared values

Render the changes in the hydration pipeline by using `kpt fn render` command:

  $ kpt fn render function-export/

    package "function-export": running function "gcr.io/kpt-fn/apply-setters:unstable": SUCCESS
    package "function-export": running function "gcr.io/kustomize-functions/example-tshirt:v0.1.0": SUCCESS
    package "function-export": rendered successfully

### Apply the package

Initialize the inventory object:

  $ kpt live init function-export/

Apply all the contents of the package recursively to the cluster

  $ kpt live apply function-export/

    service/helloworld-gke created
    deployment.apps/helloworld-gke created
    2 resource(s) applied. 2 created, 0 unchanged, 0 configured, 0 failed
    0 resource(s) pruned, 0 skipped, 0 failed


