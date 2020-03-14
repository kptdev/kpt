---
title: "Publish"
linkTitle: "Publish"
weight: 1
type: docs
description: >
    Publish a simple package of configuration
---

A kpt package is just a set of yaml files stored in a git repo. This allows
producers of kpt  packages to create or generate yaml files any way they want
using the tool of their choice.

Publishing a package is as simple as pushing a directory of Kubernetes
configuration to a git subdirectory.  The package may be versioned by
tagging the git repo with the git `package-subdirectory/package-version`
(directory scoped versioning) or as `package-version` (repo scoped versioning).

Multiple packages may exist in a single repo under separate subdirectories.
It is possible to nest packages and fetching either parent or child directories
to fetch the composite or component packages.

1. Author a package by creating resource configuration
2. Publish the package by pushing it to a git subdirectory

## Authoring a package

Package resource configuration may be bootstrapped in many different ways:

- generated from cli tools
  - `kubectl create deployment nginx --image nginx -o yaml --dry-run > deploy.yaml`
- copied from examples or blog posts
- generated from DSLs or templates

## Publishing a package

```sh
$ git clone <repo url> # or create a new repo
$ cd repository
$ mkdir nginx
# optional -- init the package with kpt metadata
$ kpt pkg init nginx --tag kpt.dev/app=nginx --description "kpt nginx package"
$ curl https://raw.githubusercontent.com/kubernetes/website/master/content/en/examples/controllers/nginx-deployment.yaml --output nginx/nginx-deployment.yaml
$ git add .
$ git commit -m "Add nginx package"
$ git tag v0.1.0
$ git push
```
