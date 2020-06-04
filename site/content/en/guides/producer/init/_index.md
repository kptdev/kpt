---
title: "Init"
linkTitle: "Init"
weight: 1
type: docs
description: >
    Initialize and publish a new package
---

A kpt package is published as a git subdirectory containing configuration
files (YAML). Publishes of kpt packages can create or generate YAML files
however they like using the tool of their choice.

Publishing a package is done by pushing the git directory
(and optionally tagging it with a version).

{{% pageinfo color="primary" %}}
Multiple packages may exist in a single repo under separate subdirectories.

Packages may be nested -- both parent (composite) and child
(component) directories may be fetched as a kpt package.

A package is versioned by tagging the git repo as one of:

- `package-subdirectory/package-version` (directory scoped versioning)
-  `package-version` (repo scoped versioning)
{{% /pageinfo %}}

{{< svg src="images/producer-guide" >}}

## Steps

1. [Create a git repo](#create-a-git-repo)
2. [Create the package contents](#create-the-package)
2. [Create configuration](#create-configuration)
3. [Publish package to git](#publish-package-to-git)


## Create a git repo

```sh
git clone REPO_URL # or create a new repo with `git init`
cd REPO_NAME
```

## Create the package

```sh
mkdir nginx
```

Recommended: initialize the package with metadata

```sh
kpt pkg init nginx --tag kpt.dev/app=nginx --description "kpt nginx package"
```

## Create configuration

```sh
$ curl https://raw.githubusercontent.com/kubernetes/website/master/content/en/examples/controllers/nginx-deployment.yaml --output nginx/nginx-deployment.yaml
```

## Publish package to git

```sh
git add .
git commit -m "Add nginx package"
```

Recommended: tag the commit as a release

```sh
# tag as DIR/VERSION for per-directory versioning
git tag nginx/v0.1.0
git push nginx/v0.1.0
```
