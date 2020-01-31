
# How to create a kpt package

A kpt package is just a set of yaml files stored in a git repo. This allows producers of kpt 
packages to create or generate yaml files any way they want using the tool of their choice. 
Once a package is ready, publishing it simply involves pushing it to git and tag it using semver 
versioning. It is possible to have multiple packages in a single repo and it is possible to nest 
packages.

```
$ git clone <repo url> # or create a new repo
$ cd repository
$ mkdir wordpress
$ kpt pkg init wordpress --tag kpt.dev/app=wordpress --description "kpt wordpress package"
# add wordpress manifests to the folder
$ git add .
$ git commit -m "Add wordpress package"
$ git tag v0.1.0
$ git push
```

## Customize a package

kpt provides several commands that can be helpful during the process of creating
and updating kpt packages.

To list all the resources in your package, use `kpt cfg tree wordpress`.\
To show the manifests for all the resources in the package, use `kpt cfg cat wordpress`

To make it easy for consumers to customize the package, it is possible to
add setters:\
`$ kpt cfg create-setter wordpress image 5.3.2-php7.2-apache --type "string" --field "image"`\
This makes it possible for consumers to easily customize a package. 

It is also possible to list all the available setters in a package:\
`$ kpt cfg list-setters wordpress`
