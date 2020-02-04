# Creating a package

This tutorial walks through the workflow of creating a package.

A kpt package is just a set of yaml files stored in a git repo. This allows producers of kpt 
packages to create or generate yaml files any way they want using the tool of their choice. 
Once a package is ready, publishing it simply involves pushing it to git and tagging it.

It is possible to have multiple packages in a single repo under separate subdirectories,
and it is possible to nest packages.

1. Author a package by creating resource configuration yaml
2. Publish the package by pushing it to a git subdirectory
3. (Optional) Create setters for package consumers to programatically modify fields

## Authoring a package

Package resource configuration may be bootstrapped in many different ways:

- generated from cli tools
  - `kubectl create deployment nginx --image nginx -o yaml --dry-run > deploy.yaml`
- copied from examples or blog posts
- generated from DSLs or templates

## Publishing a package

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

## Publishing setters

Package authors may publish custom field setters for package consumers to
programatically set specific fields using [setters](../cfg/create-setter.md)

