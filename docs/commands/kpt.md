## kpt

  Git based configuration package manager.

### Synopsis

  Git based configuration package manager.

**Packages are composed of Resource configuration** (rather than DSLs, templates, etc), but may
also contain supplemental non-Resource artifacts (e.g. README.md, arbitrary other files).

**Any existing git subdirectory containing Resource configuration** may be used as a package.

  Nothing besides a git directory containing Resource configuration is required.
  For instance, the upstream [https://github.com/kubernetes/examples/staging/cockroachdb] may
  be used as a package:

    # fetch the examples cockroachdb directory as a package
    kpt get https://github.com/kubernetes/examples/staging/cockroachdb my-cockroachdb

**Packages should use git references for versioning**.

  Package authors should use semantic versioning when publishing packages.

    # fetch the examples cockroachdb directory as a package
    kpt get https://github.com/GoogleContainerTools/kpt/examples/cockroachdb@v0.1.0 my-cockroachdb

**Packages may be modified or customized in place**.

  It is possible to directly modify the fetched package.  Some packages may expose *field setters*
  used by kustomize to change fields.  Kustomize functions may also be applied to the local
  copy of the package.

    export KUSTOMIZE_ENABLE_ALPHA_COMMANDS=true

    kpt get https://github.com/GoogleContainerTools/kpt/examples/cockroachdb my-cockroachdb
    kustomize config set my-cockroachdb/ replicas 5

**The same package may be fetched multiple times** to separate locations.

  Each instance may be modified and updated independently of the others.

    export KUSTOMIZE_ENABLE_ALPHA_COMMANDS=true

    # fetch an instance of a java package
    kpt get https://github.com/GoogleContainerTools/kpt/examples/java my-java-1
    kustomize config set my-java-1/ image gcr.io/example/my-java-1:v3.0.0

    # fetch a second instance of a java package
    kpt get https://github.com/GoogleContainerTools/kpt/examples/java my-java-2
    kustomize config set my-java-2/ image gcr.io/example/my-java-2:v2.0.0

**Packages may pull in upstream updates** from the package origin in git.

 Specify the target version to update to, and an (optional) update strategy for how to apply the
 upstream changes.

    export KUSTOMIZE_ENABLE_ALPHA_COMMANDS=true

    kpt get https://github.com/GoogleContainerTools/kpt/examples/cockroachdb my-cockroachdb
    kustomize config set my-cockroachdb/ replicas 5
    kpt update my-cockroachdb@v1.0.1 --strategy=resource-merge

## Installation

    # install kustomize
    go install sigs.k8s.io/kustomize/kustomize/v3

    # install kpt
    go install github.com/GoogleContainerTools/kpt

#### Layering and Composition

`kpt` packages are designed to compose the opinions of multiple teams within an organization,
and to unify them within individual Resources.  Packages are extended by applying additional
opinions to the base package.  The Resource fields *may* be annotated with their origins.

    # Deployment unifying the opinions of platform, petclinic-dev and app-sre teams
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: petclinic-frontend
      namespace: petclinic-prod # {"setBy":"app-sre"}
      labels:
        app: petclinic-frontend # {"setBy":"petclinic-dev"}
        env: prod # {"setBy":"app-sre"}
    spec:
      replicas: 3 # {"setBy":"app-sre"}
      selector:
        matchLabels:
          app: petclinic-frontend # {"setBy":"petclinic-dev"}
          env: prod # {"setBy":"app-sre"}
      template:
        metadata:
          labels:
            app: petclinic-frontend # {"setBy":"petclinic-dev"}
            env: prod # {"setBy":"app-sre"}
      spec:
          containers:
          - name: petclinic-frontend
            image: gcr.io/petclinic/frontend:1.7.9 # {"setBy":"app-sre"}
            args:
            - java # {"setBy":"platform"}
            - -XX:+UnlockExperimentalVMOptions # {"setBy":"platform"}
            - -XX:+UseCGroupMemoryLimitForHeap # {"setBy":"platform","description":"dynamically determine heap size"}
            ports:
            - name: http
              containerPort: 80 # {"setBy":"platform"}

#### Templates and DSLs

Note: If the use of Templates or DSLs is strongly desired, they can be fully expanded into
Resource configuration to be used as a kpt package.  These artifacts used to generated
Resource configuration may be included in the package as supplements.


#### Env Vars

  COBRA_SILENCE_USAGE
  
    Set to true to silence printing the usage on error
    
  COBRA_STACK_TRACE_ON_ERRORS
  
    Set to true to print a stack trace on an error
    
  KPT_NO_PAGER_HELP

    Set to true to print the help to the console directly instead of through
    a pager (e.g. less)
