hello-composite-pkg
==================================================

# NAME

  hello-composite-pkg

# SYNOPSIS
  
  This is an example of a kpt composite package which has nested subpackages in it. 
  Here are the steps to get, view, set and apply the package contents
  
  ### Fetch a remote package
  Get the example composite package on to local using `kpt pkg get`
  
  $ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/hello-composite-pkg \
    hello-composite-pkg
  
      fetching package /package-examples/hello-composite-pkg from https://github.com/GoogleContainerTools/kpt to hello-composite-pkg
  
  ### View the package contents
  List the package contents in a tree structure.
  
  $ kpt cfg tree hello-composite-pkg/
  
      hello-composite-pkg
      ├── [Kptfile]  Kptfile hello-composite-pkg
      ├── [deploy.yaml]  Deployment YOURSPACE/hello-composite
      └── Pkg: hello-subpkg
          ├── [Kptfile]  Kptfile hello-subpkg
          ├── [deploy.yaml]  Deployment YOURSPACE/hello-sub
          ├── hello-dir
          │   └── [configmap.yaml]  ConfigMap YOURSPACE/hello-cm
          └── Pkg: hello-nestedpkg
              ├── [Kptfile]  Kptfile hello-nestedpkg
              └── [deploy.yaml]  Deployment YOURSPACE/hello-nested
  
  The fetched package contains setter parameters which can be used to set configuration
  values from the commandline, list them
  
  $ kpt cfg list-setters hello-composite-pkg/
  
      hello-composite-pkg/
               NAME                VALUE        SET BY   DESCRIPTION   COUNT   REQUIRED
        gcloud.core.project   YOUR_PROJECT_ID                          1       No
        image                 helloworld-gke                           1       No
        namespace             YOURSPACE                                1       Yes
        tag                   0.1.0                                    1       No
      
      hello-composite-pkg/hello-subpkg/
               NAME                VALUE        SET BY   DESCRIPTION   COUNT   REQUIRED
        gcloud.core.project   YOUR_PROJECT_ID                          1       No
        image                 helloworld-gke                           1       No
        namespace             YOURSPACE                                2       Yes
        tag                   0.1.0                                    1       No
      
      hello-composite-pkg/hello-subpkg/hello-nestedpkg/
               NAME                VALUE        SET BY   DESCRIPTION   COUNT   REQUIRED
        gcloud.core.project   YOUR_PROJECT_ID                          1       No
        image                 helloworld-gke                           1       No
        namespace             YOURSPACE                                1       Yes
        tag                   0.1.0                                    1       No
        
   If you have `gcloud` set up on your local, you can observe that the value of the setter
   `gcloud.core.project` is set automatically when the package is fetched.  
   `gcloud` config setters are automatically set deriving the values from the output of
   `gcloud config list` command, when the package is fetched using [kpt pkg get].
  
  ### Provide the setter values
  `set` operation modifies the resource configuration in place by reading the resources,
  changing parameter values, and writing them back.
  
  In `list-setters` output, `namespace` setter is marked as required by the package
  publisher, hence it is mandatory to set it to a new value. You may set other setter
  parameters, either selectively in few of the packages or in all the them.
  
  `--recurse-subpackages(-R)` flag is `default:false` for set command. If not invoked,
  the set operation is performed only on the resource files of parent package and not
  the subpackages.
  
  $ kpt cfg set hello-composite-pkg/ namespace myspace -R
  
      hello-composite-pkg/
      set 1 field(s)
      
      hello-composite-pkg/hello-subpkg/
      set 2 field(s)
      
      hello-composite-pkg/hello-subpkg/hello-nestedpkg/
      set 1 field(s)
  
  ### Apply the composite package
  Create the namespace `myspace` in which resources will be deployed
  
  $ kubectl create namespace myspace
    
      namespace/myspace2 created
  
  Apply all the contents of the package recursively to the cluster
  
  $ kubectl apply -f hello-composite-pkg/ -R

      deployment.apps/hello-composite created
      deployment.apps/hello-sub created
      configmap/hello-cm created
      deployment.apps/hello-nested created
