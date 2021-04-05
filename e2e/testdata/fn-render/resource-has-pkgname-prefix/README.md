wordpress
==================================================

# NAME

  wordpress

# SYNOPSIS
  
  This is an example of a kpt package which has a subpackage in it. 
  Here are the steps to get, view, set and apply the package contents.
  
  ### Fetch a remote package
  Get the example package on to local using `kpt pkg get`
  
  $ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/subpackages/wordpress
  
      fetching package /package-examples/subpackages/wordpress from https://github.com/GoogleContainerTools/kpt to wordpress
  
  ### View the package contents
  List the package contents in a tree structure.
  
  $ kpt cfg tree wordpress/
  
      wordpress
      ├── [wordpress-deployment.yaml]  Deployment wordpress
      ├── [wordpress-deployment.yaml]  Service wordpress
      ├── [wordpress-deployment.yaml]  PersistentVolumeClaim wp-pv-claim
      └── Pkg: mysql
          ├── [mysql-deployment.yaml]  PersistentVolumeClaim mysql-pv-claim
          ├── [mysql-deployment.yaml]  Deployment wordpress-mysql
          └── [mysql-deployment.yaml]  Service wordpress-mysql 
  
  ### Provide the setter values
  Provide the values for all the setters.
  
  $ kpt fn render wordpress/
  
  ### Apply the package
  
  Apply all the contents of the package recursively to the cluster
  
  $ kubectl apply -f wordpress/ -R

      service/wordpress-mysql created
      persistentvolumeclaim/mysql-pv-claim created
      deployment.apps/wordpress-mysql created
      service/wordpress created
      persistentvolumeclaim/wp-pv-claim created
      deployment.apps/wordpress created
      
[tree]: https://googlecontainertools.github.io/kpt/reference/cfg/tree/
