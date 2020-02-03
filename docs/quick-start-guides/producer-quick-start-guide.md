
# How to create a kpt package

A kpt package is just a set of yaml files stored in a git repo. This allows producers of kpt 
packages to create or generate yaml files any way they want using the tool of their choice. 
Once a package is ready, publishing it simply involves pushing it to git and tag it using semver 
versioning. It is possible to have multiple packages in a single repo and it is possible to nest 
packages.

```
$ git clone <repo url> # or create a new repo
$ cd repository
$ mkdir nginx
$ kpt pkg init nginx --tag kpt.dev/app=nginx --description "kpt nginx package"
$ curl https://raw.githubusercontent.com/kubernetes/website/master/content/en/examples/controllers/nginx-deployment.yaml --output nginx/nginx-deployment.yaml
$ git add .
$ git commit -m "Add nginx package"
$ git tag v0.1.0
$ git push
```

## Customize a package

kpt provides several commands that can be helpful during the process of creating
and updating kpt packages.

To list all the resources in your package, use `kpt cfg tree nginx`.\
To show the manifests for all the resources in the package, use `kpt cfg cat nginx`

To make it easy for consumers to customize the package, it is possible to
add setters:\
`$ kpt cfg create-setter nginx image nginx:1.7.9 --type "string" --field "image"`\
This makes it possible for consumers to easily customize a package. 

It is also possible to list all the available setters in a package:\
`$ kpt cfg list-setters nginx`
