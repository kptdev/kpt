---
title: "Publish a package with subpackages"
linkTitle: "Subpackages"
weight: 7
type: docs
description: >
  Create and publish a kpt package with subpackage in its directory tree
---

{{% pageinfo color="warning" %}}

#### Notice: Subpackages support is available with kpt version v0.34.0+ for [cfg] commands only

{{% /pageinfo %}}

This guide walks you through an example to create, parameterize and publish a
kpt package with a [subpackage] in it.

## Steps

1. [Create the package](#create-the-package)
2. [Add setters and substitutions](#add-setters-and-substitutions)
3. [Publish the package](#publish-the-package)

## Create the package

Initialize a `kpt` package with a [subpackage] in its directory tree.

```sh
mkdir wordpress
kpt pkg init wordpress

mkdir wordpress/mysql
kpt pkg init wordpress/mysql
```

Download the `Wordpress` deployment configuration file.

```sh
cd wordpress
curl -LO https://k8s.io/examples/application/wordpress/wordpress-deployment.yaml
```

Download the `MySQL` deployment configuration file.

```sh
cd mysql
curl -LO https://k8s.io/examples/application/wordpress/mysql-deployment.yaml
cd ../../
```

## Add setters and substitutions

### Add annotations

As a package publisher you might want to add [annotations] to kubernetes objects.
You can leverage `--recurse-subpackages(-R)` flag to create it recursively in both
`wordpress` and `mysql` packages.

```sh
kpt cfg annotate wordpress/ --kv teamname=YOURTEAM -R
```

Output:

```sh
wordpress/
added annotations in the package

wordpress/mysql/
added annotations in the package
```

Similarly add `projectId` annotation to both the packages.

```sh
kpt cfg annotate wordpress/ --kv projectId=PROJECT_ID -R
```

Output:

```sh
wordpress/
added annotations in the package

wordpress/mysql/
added annotations in the package
```

### Create setters

Create [Setters] for the annotations values which you just added

```sh
kpt cfg create-setter wordpress/ teamname YOURTEAM -R --required
```

Output:

```sh
wordpress/
created setter "teamname"

wordpress/mysql/
created setter "teamname"
```

Similarly create an [auto-setter] with name `gcloud.core.project`.

```sh
kpt cfg create-setter wordpress/ gcloud.core.project PROJECT_ID -R
```

Output:

```sh
wordpress/
created setter "gcloud.core.project"

wordpress/mysql/
created setter "gcloud.core.project"
```

### Create substitutions

Create [Substitutions] so that package consumers can substitute values,
using `kpt cfg set`

```sh
kpt cfg create-subst wordpress image-tag \
--field-value wordpress:4.8-apache \
--pattern \${image}:\${tag}-apache
```

Output:

```sh
wordpress/
unable to find setter with name image, creating new setter with value wordpress
unable to find setter with name tag, creating new setter with value 4.8
created substitution "image-tag"
```

```sh
kpt cfg create-subst wordpress/mysql image-tag \
--field-value mysql:5.6 \
--pattern \${image}:\${tag}
```

Output:

```sh
wordpress/mysql/
unable to find setter with name image, creating new setter with value wordpress
unable to find setter with name tag, creating new setter with value 4.8
created substitution "image-tag"
```

### List and verify setters/substitutions

Use list-setters command to verify that the setters and substitutions are created as expected

```sh
kpt cfg list-setters wordpress/ --include-subst
```

Output:

```sh
wordpress/
         NAME             VALUE      SET BY   DESCRIPTION   COUNT   REQUIRED
  gcloud.core.project   PROJECT_ID                          3       No
  image                 wordpress                           1       No
  tag                   4.8                                 1       No
  teamname              YOURTEAM                            3       Yes
--------------- ------------------------ --------------
  SUBSTITUTION          PATTERN           REFERENCES
  image-tag      ${image}:${tag}-apache   [image,tag]

wordpress/mysql/
         NAME             VALUE      SET BY   DESCRIPTION   COUNT   REQUIRED
  gcloud.core.project   PROJECT_ID                          3       No
  image                 wordpress                           1       No
  tag                   4.8                                 1       No
  teamname              YOURTEAM                            3       Yes
--------------- ----------------- --------------
  SUBSTITUTION       PATTERN       REFERENCES
  image-tag      ${image}:${tag}   [image,tag]
```

## Publish the package

Now that as a package creator, you have created and parameterized a `kpt` package,
publish it so that package consumers can consume it.

Create a [git repo] in your profile with name `wordpress`

```sh
cd wordpress/
git init; git add .; git commit -am "Publish package";
git remote add origin <YOUR_GIT_REPO_LINK>
git push origin master
```

## Next steps

Go through the [consumer guide] to consume the published package

[kpt pkg get]: ../../../reference/pkg/get/
[substitutions]: https://googlecontainertools.github.io/kpt/guides/producer/substitutions/
[git repo]: https://docs.github.com/en/enterprise/2.13/user/articles/creating-a-new-repository
[set]: https://googlecontainertools.github.io/kpt/guides/consumer/set/
[setters]: https://googlecontainertools.github.io/kpt/guides/producer/setters/
[auto-setter]: https://googlecontainertools.github.io/kpt/guides/producer/setters/#auto-setters
[cfg]: https://googlecontainertools.github.io/kpt/reference/cfg/
[subpackage]: https://googlecontainertools.github.io/kpt/concepts/packaging/#subpackages
[consumer guide]: https://googlecontainertools.github.io/kpt/guides/producer/subpackages/
