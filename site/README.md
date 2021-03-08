+++
title = "Kpt"
linkTitle = "Kpt"

+++

{{% blocks/lead color="primary" %}}
### Overview

Kpt (pronounced “kept”) is an OSS tool for building declarative workflows
on top of resource configuration.

Its git + YAML architecture means it just works with existing tools,
frameworks, and platforms.

Kpt includes solutions to fetch, display, customize, update, validate, and
apply Kubernetes configuration.

{{% /blocks/lead %}}

{{< blocks/section color="light" >}}
{{% blocks/feature icon="fa fa-download" title="Install" url="installation/" %}}
### Installation
[Install](installation/) via gcloud, homebrew, binaries or source.
{{% /blocks/feature %}}

{{% blocks/feature icon="fab fa-github" title="Contribute" url="https://github.com/GoogleContainerTools/kpt/blob/master/CONTRIBUTING.md" %}}
### Contributing
We use a pull request workflow on [**GitHub**](https://github.com/GoogleContainerTools/kpt/blob/master/CONTRIBUTING.md). New users are always welcome!
{{% /blocks/feature %}}

{{< /blocks/section >}}
### Features

{{< blocks/section color="dark" >}}
{{% blocks/feature title="`kpt pkg`" url="reference/pkg/" %}}
#### [kpt pkg](reference/pkg/)
A packaging solution for resource configuration.
Fetch and update configuration using git and YAML.
{{% /blocks/feature %}}

{{% blocks/feature title="`kpt live`" url="reference/live/" %}}
#### [kpt live](reference/live/)
The next-generation of apply with manifest based pruning and resource
status.
{{% /blocks/feature %}}

{{% blocks/feature title="`kpt fn`" url="reference/fn/" %}}
#### [kpt fn](reference/fn/)
Extend the built-in capabilities of kpt by writing functions to generate,
transform and validate configuration.
{{% /blocks/feature %}}

{{< /blocks/section >}}
