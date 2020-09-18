wordpress
==================================================

# NAME

  wordpress

# SYNOPSIS
  
  This is an example of a kpt package which has a subpackage in it. 
  Here are the steps to get, view, set and apply the package contents
  
  ### Fetch a remote package
  Get the example package on to local using `kpt pkg get`
  
  $ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/subpackages/wordpress \
    wordpress
  
      fetching package /package-examples/subpackages/wordpress from https://github.com/GoogleContainerTools/kpt to wordpress
  
  ### View the package contents
  List the package contents in a tree structure.
  
  $ kpt cfg tree wordpress/
  
      wordpress
      ├── [Kptfile]  Kptfile wordpress
      ├── [wordpress-deployment.yaml]  Deployment wordpress
      ├── [wordpress-deployment.yaml]  Service wordpress
      ├── [wordpress-deployment.yaml]  PersistentVolumeClaim wp-pv-claim
      └── Pkg: mysql
          ├── [Kptfile]  Kptfile mysql
          ├── [mysql-deployment.yaml]  PersistentVolumeClaim mysql-pv-claim
          ├── [mysql-deployment.yaml]  Deployment wordpress-mysql
          └── [mysql-deployment.yaml]  Service wordpress-mysql
  
  The fetched package contains [setters]. Invoke [list-setters] command to list
  the [setters] recursively in all the packages.
  
  $ kpt cfg list-setters wordpress/
  
      wordpress/
               NAME             VALUE      SET BY   DESCRIPTION   COUNT   REQUIRED  
        gcloud.core.project   PROJECT_ID                          3       No        
        image                 wordpress                           1       No        
        tag                   4.8                                 1       No        
        teamname              YOURTEAM                            3       Yes       
      
      wordpress/mysql/
               NAME             VALUE      SET BY   DESCRIPTION   COUNT   REQUIRED  
        gcloud.core.project   PROJECT_ID                          3       No        
        image                 mysql                               1       No        
        tag                   5.6                                 1       No        
        teamname              YOURTEAM                            3       Yes       
        
   You may notice that the [auto-setter] `gcloud.core.project` is already set if you
   have `gcloud` configured on your local.
  
  ### Provide the setter values
  Provide the values for all the [required setters]. By default, [set] 
  command is performed only on the resource files of provided package and not its 
  subpackages. `--recurse-subpackages(-R)` can be leveraged to run the command on 
  subpackages recursively.
  
  $ kpt cfg set wordpress/ teamname myteam -R
  
      wordpress/
      set 3 field(s)
      
      wordpress/mysql/
      set 3 field(s)
  
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
[auto-setter]: https://googlecontainertools.github.io/kpt/guides/producer/setters/#auto-setters
[subpackage]: https://googlecontainertools.github.io/kpt/concepts/packaging/#subpackages
[setters]: https://googlecontainertools.github.io/kpt/guides/producer/setters/
[set]: https://googlecontainertools.github.io/kpt/reference/cfg/set/
[required setters]: https://googlecontainertools.github.io/kpt/guides/producer/setters/#marking-a-field-as-required
[list-setters]: https://googlecontainertools.github.io/kpt/reference/cfg/list-setters/
