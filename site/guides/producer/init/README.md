---
title: "Init"
linkTitle: "Init"
weight: 1
type: docs
description: >
    Initialize and publish a new package
---

{{% hide %}}
<!-- @makeWorkplace @verifyGuides-->
```
# Set up workspace for the test.
TEST_HOME=$(mktemp -d)
cd $TEST_HOME
```
{{% /hide %}}

A kpt package is published as a git subdirectory containing configuration
files (YAML). Publishers of kpt packages can create or generate YAML files
however they like using the tool of their choice.

Publishing a package is done by pushing the git directory
(and optionally [tagging] it with a version).

{{% pageinfo color="primary" %}}
Multiple packages may exist in a single repo under separate subdirectories.

Packages may be nested -- both parent (composite) and child
(component) directories may be fetched as a kpt package.

A package is versioned by tagging the git repo as one of:

- `package-subdirectory/package-version` (directory scoped versioning)
- `package-version` (repo scoped versioning)

So package example that exists in the example folder of the repo, can
be individually versioned (as version v1.0.2) by creating the tag `example/v1.0.2`.

{{% /pageinfo %}}

![img](/static/images/producer-guide.svg)

## Steps

1. [Create a git repo](#create-a-git-repo)
2. [Create the package contents](#create-the-package)
3. [Create configuration](#create-configuration)
4. [Publish package to git](#publish-package-to-git)
5. [Fetch the released package](#fetch-the-released-package)

## Create a git repo

<!-- @defineEnvVars @verifyGuides-->
```sh
REPO_NAME=my-repo
REPO_URL="<url>"
```

{{% hide %}}
<!-- @setRepoUrlForTest @verifyGuides-->
```
# Set up workspace for the test.
REPO_URL=file://$(pwd)/$REPO_NAME.git
```
{{% /hide %}}

<!-- @setupRepo @verifyGuides-->
```sh
mkdir $REPO_NAME # or clone with git `git clone`
git init $REPO_NAME # only if new repo
```

## Create the package

<!-- @createPackage @verifyGuides-->
```sh
mkdir $REPO_NAME/nginx
```

Recommended: initialize the package with metadata

<!-- @initPackage @verifyGuides-->
```sh
kpt pkg init $REPO_NAME/nginx --tag kpt.dev/app=nginx --description "kpt nginx package"
```

## Create configuration

<!-- @addConfig @verifyGuides-->
```sh
curl https://raw.githubusercontent.com/kubernetes/website/master/content/en/examples/controllers/nginx-deployment.yaml --output $REPO_NAME/nginx/nginx-deployment.yaml
```

## Publish package to git

<!-- @commitRepo @verifyGuides-->
```sh
(cd $REPO_NAME && git add . && git commit -m "Add nginx package")
```

Recommended: tag the commit as a release

<!-- @createTag @verifyGuides-->
```sh
# tag as DIR/VERSION for per-directory versioning
(cd $REPO_NAME && git tag nginx/v0.1.0)
# git push nginx/v0.1.0 # requires an upstream repo
```

## Fetch the released package

<!-- @fetchPackage @verifyGuides-->
```sh
kpt pkg get $REPO_URL/nginx@v0.1.0 nginx
```

{{% hide %}}
<!-- @setRepoUrlForTest @verifyGuides-->
```
grep "ref: v0.1.0" nginx/Kptfile
grep "kpt nginx package" nginx/README.md
grep  "name: nginx-deployment" nginx/nginx-deployment.yaml
```
{{% /hide %}}

[tagging]: https://git-scm.com/book/en/v2/Git-Basics-Tagging
