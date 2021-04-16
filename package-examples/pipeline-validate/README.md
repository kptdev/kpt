# pipeline-validate

This example package demonstrates how you can modify the config and ensure 
that modifications are compliant with the policies. This package uses:

1. a `mutator` function called `label-namespace` to customize (or modify) the config
2. a `validator` function `gatekeeper` to ensure changes are inline with the policy 

Putting a validation function into your package allows you to give package
consumers instant feedback on whether their customization violates config
policy.

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

  $ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/pipeline-validate

    fetching package /package-examples/pipeline-validate from https://github.com/GoogleContainerTools/kpt to pipeline-validate

### View the package contents

List the package contents in a tree structure.

  $ kpt pkg tree pipeline-validate/

    PKG: pipeline-validate
    ├── [Kptfile]  Kptfile kpt-pipeline-validate-example
    └── resources
        ├── [resources.yaml]  Namespace development
        ├── [resources.yaml]  Deployment development/nginx-deployment
        └── constraints
            ├── [deployment-must-have-owner.yaml]  K8sRequiredLabels deployment-must-have-owner
            └── [requiredlabels.yaml]  ConstraintTemplate k8srequiredlabels

### Configure functions

The package contains a function pipeline in the `Kptfile` which has
one `set-label` and `gatekeeper-validate` functions.  
The `set-label` function allows you to set one or more labels to any
resource in your configuration.  In this case it's namespace label as
defined by the fieldSpecs.  The `gatekeeper-validate` function allows 
you to use gatekeeper for checks on the configuration.

  pipeline:
    mutators:
      - image: gcr.io/kpt-fn/set-label:unstable
        config:
          apiVersion: fn.kpt.dev/v1alpha1
          kind: SetLabelConfig
          metadata:
            name: label-color-blue
          labels:
            color: blue
          fieldSpecs:
            - kind: Namespace
              create: true        
    validators:
      - image: gcr.io/kpt-functions/gatekeeper-validate



### Render the declared values

Render the changes in the rendering pipeline by using `kpt fn render` command:

  $ kpt fn render pipeline-validate/

    package "pipeline-validate": running function "gcr.io/kpt-fn/set-label:unstable": SUCCESS
    package "pipeline-validate": running function "gcr.io/kpt-functions/gatekeeper-validate": SUCCESS
    package "pipeline-validate": rendered successfully


If you remove the owner label from `resources.yaml` and re-run the rendering
you should see an error:


  $ kpt fn render pipeline-validate/

    package "pipeline-validate": running function "gcr.io/kpt-fn/set-label:unstable": SUCCESS
    package "pipeline-validate": running function "gcr.io/kpt-functions/gatekeeper-validate": FAILED
    fn.render: pkg pipeline-validate:
            pkg.render:
            pipeline.run: Error: Found 1 violations:

    [1] Deployment objects should have an 'owner' label indicating who created them.

    name: "nginx-deployment"
    path: resources/resources.yaml


### Apply the package

Initialize the inventory object:

  $ kpt live init pipeline-validate/

Apply all the contents of the package recursively to the cluster

  $ kpt live apply pipeline-validate/

    TODO: getting error: can't find scope for resource K8sRequiredLabels.constraints.gatekeeper.sh deployment-must-have-owner
