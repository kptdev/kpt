+++
title = "Kpt"
linkTitle = "Kpt"

+++

{{% blocks/lead color="primary" %}}
Kpt (pronounced “kept”) is an OSS tool for building declarative workflows
on top of resource configuration.

Its git + YAML architecture means it just works with existing tools,
frameworks, and platforms.

Kpt includes solutions to fetch, display, customize, update, validate, and
apply Kubernetes configuration.

{{% /blocks/lead %}}

{{< blocks/section color="dark" >}}
{{% blocks/feature title="`kpt pkg`" url="reference/pkg" %}}
A packaging solution for resource configuration.
Fetch and update configuration using git and YAML.
{{% /blocks/feature %}}

{{% blocks/feature title="`kpt cfg`" url="reference/cfg" %}}
A cli UX layer on top of YAML
Display and modify configuration files without ever dropping into an editor.
{{% /blocks/feature %}}

{{% blocks/feature title="`kpt live`" url="reference/live" %}}
The next-generation of apply with manifest based pruning and resource
status.
{{% /blocks/feature %}}

{{% blocks/feature title="`kpt fn`" url="reference/fn" %}}
Extend the built-in capabilities of kpt by writing functions to generate,
transform and validate configuration.
{{% /blocks/feature %}}

{{< /blocks/section >}}

{{< blocks/section >}}
{{% blocks/feature icon="fab fa-envelope" title="Install" url="installation" %}}
Install via gcloud, homebrew, binaries or source.
{{% /blocks/feature %}}

{{% blocks/feature icon="fab fa-github" title="Contribute" %}}
We do a [Pull Request](https://github.com/GoogleContainerTools/kpt/pulls) contributions workflow on **GitHub**. New users are always welcome!
{{% /blocks/feature %}}

{{< /blocks/section >}}


